// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package home

import (
	"sort"
	"strconv"
	"strings"
)

type (
	Home struct {
		zonesByName map[string]Zone
	}
	Zone struct {
		name          string
		devicesByName map[string]Device
	}
	Device struct {
		name           string
		controlsByName map[string]Control
	}

	Control interface {
		Name() string
		Topic() string
	}

	Enum struct {
		name   string
		topic  string
		Value  string
		Values []string
	}

	Range struct {
		name  string
		topic string
		Value int
		Min   int
		Max   int
	}

	Toggle struct {
		name  string
		topic string
		Value bool
	}
)

func OfValuesByTopic(valuesByTopic map[string]string) Home {
	zonesByName := map[string]Zone{}

	insertControl := func(zone, device, control string, c Control) {
		if _, ok := zonesByName[zone]; !ok {
			zonesByName[zone] = Zone{
				name:          zone,
				devicesByName: map[string]Device{},
			}
		}
		devicesByName := zonesByName[zone].devicesByName
		if _, ok := devicesByName[device]; !ok {
			devicesByName[device] = Device{
				name:           device,
				controlsByName: map[string]Control{},
			}
		}
		controlsByName := devicesByName[device].controlsByName
		controlsByName[control] = c
	}

	for topic, v := range valuesByTopic {
		if v == "" {
			continue
		}
		parts := strings.Split(topic, "/")
		if len(parts) != 4 {
			continue
		}
		zone, device, control := parts[1], parts[2], parts[3]

		switch {
		case control == "power":
			on := false
			if v == "on" {
				on = true
			}
			insertControl(zone, device, control, &Toggle{
				name:  "power",
				topic: topic,
				Value: on,
			})
		case strings.HasSuffix(control, "_percent"):
			value, err := strconv.Atoi(v)
			if err != nil {
				value = 0
			}
			insertControl(zone, device, control, &Range{
				name:  strings.TrimSuffix(control, "_percent"),
				topic: topic,
				Value: value,
				Min:   0,
				Max:   100,
			})
		case strings.HasSuffix(control, "_degrees"):
			value, err := strconv.Atoi(v)
			if err != nil {
				value = 0
			}
			insertControl(zone, device, control, &Range{
				name:  strings.TrimSuffix(control, "_degrees"),
				topic: topic,
				Value: value,
				Min:   0,
				Max:   360,
			})
		case control == "kelvin":
			// TODO: Un-magic kelvin.
			value, err := strconv.Atoi(v)
			if err != nil {
				value = 2500
			}
			insertControl(zone, device, control, &Range{
				name:  control,
				topic: topic,
				Value: value,
				Min:   2500,
				Max:   9000,
			})
		case strings.HasSuffix(control, "_enum"):
			var values []string
			if vs, ok := valuesByTopic[topic+"/values"]; ok {
				values = strings.Split(vs, "\n")
			}
			insertControl(zone, device, control, &Enum{
				name:   strings.TrimSuffix(control, "_enum"),
				topic:  topic,
				Value:  v,
				Values: values,
			})
		default:
			// Do nothing?
		}
	}
	return Home{
		zonesByName: zonesByName,
	}
}

func (h Home) Zones() []Zone {
	var zones []Zone
	for _, zone := range h.zonesByName {
		zones = append(zones, zone)
	}
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].name < zones[j].name
	})
	return zones
}

func (z Zone) Name() string {
	return z.name
}
func (z Zone) Devices() []Device {
	var devices []Device
	for _, device := range z.devicesByName {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].name < devices[j].name
	})
	return devices
}

func (d Device) Name() string {
	return d.name
}
func (d Device) Controls() []Control {
	var controls []Control
	for _, control := range d.controlsByName {
		controls = append(controls, control)
	}
	sort.Slice(controls, func(i, j int) bool {
		return controls[i].Name() < controls[j].Name()
	})
	return controls
}

func (e *Enum) Name() string {
	return e.name
}
func (e *Enum) Topic() string {
	return e.topic
}
func (r *Range) Name() string {
	return r.name
}
func (r *Range) Topic() string {
	return r.topic
}
func (t *Toggle) Name() string {
	return t.name
}
func (t *Toggle) Topic() string {
	return t.topic
}
