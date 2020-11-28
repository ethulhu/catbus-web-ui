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
	"sort"
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

	// Return the tree of zones/devices/controls under home/{path} as JSON.
	// For example,
	// 	GET /home/ => the entire home.
	// 	GET /home/bedroom => everything under home/bedroom.
	m.PathPrefix("/home/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payloadByTopicMu.RLock()
			defer payloadByTopicMu.RUnlock()
			h := home.OfValuesByTopic(payloadByTopic)

			fmt.Fprintf(w, "<html><head><title>Home</title></title>")
			fmt.Fprintf(w, "<body><h1>Home</h1>")
			zones := h.Zones()
			sort.Slice(zones, func(i, j int) bool {
				return zones[i].Name() < zones[j].Name()
			})

			for _, zone := range zones {
				fmt.Fprintf(w, "<section><h2>%s</h2>", zone.Name())
				devices := zone.Devices()
				sort.Slice(devices, func(i, j int) bool {
					return devices[i].Name() < devices[j].Name()
				})

				for _, device := range devices {
					fmt.Fprintf(w, "<section><h3>%s</h3>", device.Name())
					controls := device.Controls()
					sort.Slice(controls, func(i, j int) bool {
						return controls[i].Name() < controls[j].Name()
					})

					fmt.Fprintf(w, "<table>")
					for _, control := range controls {
						fmt.Fprintf(w, "<tr><td>%s</td><td>", control.Name())
						switch control := control.(type) {
						case *home.Enum:
							fmt.Fprintf(w, "<select>")
							for _, value := range control.Values {
								if value == control.Value {
									fmt.Fprintf(w, "<option selected>%s</option>", value)
								} else {
									fmt.Fprintf(w, "<option>%s</option>", value)
								}
							}
							fmt.Fprintf(w, "</select>")
						case *home.Range:
							fmt.Fprintf(w, "<input type='range' min='%v' max='%v' value='%v'>", control.Min, control.Max, control.Value)
						case *home.Toggle:
							if control.Value {
								fmt.Fprintf(w, "<input type='checkbox' checked>")
							} else {
								fmt.Fprintf(w, "<input type='checkbox'>")
							}
						default:
							panic("unknown control type")
						}
						fmt.Fprintf(w, "</td></tr>")
					}
					fmt.Fprintf(w, "</table>")
					fmt.Fprintf(w, "</section>")
				}
				fmt.Fprintf(w, "</section>")
			}
			fmt.Fprintf(w, "</body></html>")
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
