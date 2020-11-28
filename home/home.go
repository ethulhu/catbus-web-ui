package home

import (
	"strconv"
	"strings"
)

type (
	Home struct {
		zonesByName map[string]map[string]map[string]Control
	}

	Control interface {
		Set(value string) error
	}

	Enum struct {
		Name   string
		Value  string
		Values []string
	}

	Range struct {
		Name  string
		Value int
		Min   int
		Max   int
	}

	Toggle struct {
		Name string
		On   bool
	}
)

func OfValuesByTopic(valuesByTopic map[string]string) Home {
	zonesByName := map[string]map[string]map[string]Control{}

	insertControl := func(zone, device, control string, c Control) {
		if _, ok := zonesByName[zone]; !ok {
			zonesByName[zone] = map[string]map[string]Control{}
		}
		devicesByName := zonesByName[zone]
		if _, ok := devicesByName[device]; !ok {
			devicesByName[device] = map[string]Control{}
		}
		controlsByName := devicesByName[device]
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
				Name: "power",
				On:   on,
			})
		case strings.HasSuffix(control, "_percent"):
			value, err := strconv.Atoi(v)
			if err != nil {
				value = 0
			}
			insertControl(zone, device, control, &Range{
				Name:  strings.TrimSuffix(control, "_percent"),
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
				Name:  strings.TrimSuffix(control, "_degrees"),
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
				Name:  control,
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
				Name:   strings.TrimSuffix(control, "_enum"),
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

func (h Home) Zones() []string {
	var zones []string
	for zone := range h.zonesByName {
		zones = append(zones, zone)
	}
	return zones
}
func (h Home) Devices(zone string) []string {
	var devices []string
	if devicesByName, ok := h.zonesByName[zone]; ok {
		for device := range devicesByName {
			devices = append(devices, device)
		}
	}
	return devices
}
func (h Home) Controls(zone, device string) []string {
	var controls []string
	if devicesByName, ok := h.zonesByName[zone]; ok {
		if controlsByName, ok := devicesByName[device]; ok {
			for control := range controlsByName {
				controls = append(controls, control)
			}
		}
	}
	return controls
}
func (h Home) Control(zone, device, control string) (Control, bool) {
	if devicesByName, ok := h.zonesByName[zone]; ok {
		if controlsByName, ok := devicesByName[device]; ok {
			if c, ok := controlsByName[control]; ok {
				return c, true
			}
		}
	}
	return nil, false
}

func (e *Enum) Set(value string) error {
	panic("not implemented")
}
func (r *Range) Set(value string) error {
	panic("not implemented")
}
func (t *Toggle) Set(value string) error {
	panic("not implemented")
}
