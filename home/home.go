// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package home

import (
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
		Set(value string) error
	}

	Enum struct {
		name   string
		Value  string
		Values []string
	}

	Range struct {
		name  string
		Value int
		Min   int
		Max   int
	}

	Toggle struct {
		name  string
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

	for k, v := range valuesByTopic {
		if v == "" {
			continue
		}
		parts := strings.Split(k, "/")
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
				Value: on,
			})
		case strings.HasSuffix(control, "_percent"):
			value, err := strconv.Atoi(v)
			if err != nil {
				value = 0
			}
			insertControl(zone, device, control, &Range{
				name:  strings.TrimSuffix(control, "_percent"),
				Value: value,
				Min:   0,
				Max:   2500,
			})
		case strings.HasSuffix(control, "_degrees"):
			value, err := strconv.Atoi(v)
			if err != nil {
				value = 0
			}
			insertControl(zone, device, control, &Range{
				name:  strings.TrimSuffix(control, "_degrees"),
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
				Value: value,
				Min:   2500,
				Max:   9000,
			})
		case strings.HasSuffix(control, "_enum"):
			var values []string
			if vs, ok := valuesByTopic[k+"/values"]; ok {
				values = strings.Split(vs, "\n")
			}
			insertControl(zone, device, control, &Enum{
				name:   strings.TrimSuffix(control, "_enum"),
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
	return controls
}

func (e *Enum) Name() string {
	return e.name
}
func (e *Enum) Set(value string) error {
	panic("not implemented")
}
func (r *Range) Name() string {
	return r.name
}
func (r *Range) Set(value string) error {
	panic("not implemented")
}
func (t *Toggle) Name() string {
	return t.name
}
func (t *Toggle) Set(value string) error {
	panic("not implemented")
}
