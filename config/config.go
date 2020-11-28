// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"io/ioutil"
)

type (
	Config struct {
		BrokerURI string
	}

	config struct {
		MQTTBroker string `json:"mqttBroker"`
	}
)

func ParseFile(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := config{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return nil, err
	}

	return configFromConfig(raw), nil
}

func configFromConfig(raw config) *Config {
	c := &Config{
		BrokerURI:    raw.MQTTBroker,
	}
	return c
}
