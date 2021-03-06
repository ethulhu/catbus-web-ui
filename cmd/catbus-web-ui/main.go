// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

//go:generate go run github.com/rakyll/statik -src=static

// Binary catbus-web-ui is a web UI for Catbus.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"go.eth.moe/catbus"
	"go.eth.moe/catbus-web-ui/config"
	"go.eth.moe/catbus-web-ui/home"

	_ "go.eth.moe/catbus-web-ui/cmd/catbus-web-ui/statik"
)

var (
	port   = flag.Uint("port", 0, "port to listen on")
	socket = flag.String("socket", "", "path to socket to listen to")

	configPath = flag.String("config-path", "", "path to config.json")
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("must set -config-path")
	}
	config, err := config.ParseFile(*configPath)
	if err != nil {
		log.Fatalf("could not read config %q: %v", *configPath, err)
	}

	if (*port == 0) == (*socket == "") {
		log.Fatal("must set -socket XOR -port")
	}
	var conn net.Listener
	if *port != 0 {
		conn, err = net.Listen("tcp", fmt.Sprintf(":%v", *port))
	} else {
		_ = os.Remove(*socket)
		conn, err = net.Listen("unix", *socket)
		_ = os.Chmod(*socket, 0660)
	}
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer conn.Close()

	payloadByTopic := map[string]string{}
	payloadByTopicMu := sync.RWMutex{}
	broker := catbus.NewClient(config.BrokerURI, catbus.ClientOptions{
		ConnectHandler: func(broker catbus.Client) {
			log.Printf("connected to broker %q", config.BrokerURI)
			broker.Subscribe("home/#", func(_ catbus.Client, m catbus.Message) {
				payloadByTopicMu.Lock()
				defer payloadByTopicMu.Unlock()

				payloadByTopic[m.Topic] = m.Payload
				if m.Payload == "" {
					delete(payloadByTopic, m.Topic)
				}
			})
		},
		DisconnectHandler: func(_ catbus.Client, err error) {
			log.Printf("connected to broker %q: %v", config.BrokerURI, err)
		},
	})
	go func() {
		if err := broker.Connect(); err != nil {
			log.Fatalf("could not connect to broker %q: %v", config.BrokerURI, err)
		}
	}()

	m := mux.NewRouter()
	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("not found: %v %v %v", r.Method, r.URL, r.Form)
		if r.URL.Path != "/favicon.ico" {
			log.Print(msg)
		}
		http.Error(w, msg, http.StatusNotFound)
	})

	// Return the tree of zones/devices/controls under home/{path} as JSON.
	// For example,
	// 	GET /home/ => the entire home.
	// 	GET /home/bedroom => everything under home/bedroom.
	// TODO: maybe actually do the prefix thing?
	m.Path("/home/").
		Methods("GET").
		Headers("Accept", "application/json").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payloadByTopicMu.RLock()
			defer payloadByTopicMu.RUnlock()

			h := home.OfValuesByTopic(payloadByTopic)

			rsp := map[string]interface{}{}
			for _, zone := range h.Zones() {
				rspZone := map[string]interface{}{}
				for _, device := range zone.Devices() {
					rspDevice := map[string]interface{}{}
					for _, control := range device.Controls() {
						rspControl := map[string]interface{}{}
						switch control := control.(type) {
						case *home.Enum:
							rspControl["value"] = control.Value
							rspControl["values"] = control.Values
						case *home.Range:
							rspControl["value"] = control.Value
							rspControl["min"] = control.Min
							rspControl["max"] = control.Max
						case *home.Toggle:
							rspControl["value"] = control.Value
						default:
							panic("unknown control type")
						}
						rspDevice[control.Name()] = rspControl
					}
					rspZone[device.Name()] = rspDevice
				}
				rsp[zone.Name()] = rspZone
			}

			bytes, err := json.Marshal(rsp)
			if err != nil {
				panic(err)
			}
			w.Write(bytes)
		})

	m.Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payloadByTopicMu.RLock()
			defer payloadByTopicMu.RUnlock()
			h := home.OfValuesByTopic(payloadByTopic)

			if err := indexTmpl.Execute(w, h); err != nil {
				log.Printf("could not template: %v", err)
			}
		})

	m.Path("/home/{zone}/{device}/{control}").
		Methods("POST").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			topic := r.URL.Path[1:]
			value := r.FormValue("value")
			if value == "" {
				return
			}

			payloadByTopicMu.Lock()
			defer payloadByTopicMu.Unlock()

			if _, ok := payloadByTopic[topic]; ok {
				payloadByTopic[topic] = value // TODO: Can this de-sync from the broker?
				go broker.Publish(topic, catbus.Retain, value)
			}
		})

	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}
	m.PathPrefix("/").
		Methods("GET").
		Handler(http.FileServer(statikFS))

	log.Printf("starting HTTP server on %v", conn.Addr())
	if err := http.Serve(conn, m); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
