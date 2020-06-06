package main

import (
	"encoding/json"
	"io/ioutil"
)

type ConfigFile struct {
	ServerConfig ServerConfig
	ClientConfig ClientConfig
	Inputs       map[string]Input
	Outputs      map[string]Output
}

type ServerConfig struct {
	ListenAddress string
}

type ClientConfig struct {
	UseMDNS bool
}

type Input struct {
	Pin       string
	Hidden    bool
	Invert    bool
	OnRising  string
	OnFalling string
	Method    string
	Payload   string
	Debounce  string
}

type Output struct {
	Pin    string
	Hidden bool
	Invert bool
	Pulse  string
}

func loadConfig(file string) (ConfigFile, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return ConfigFile{}, err
	}

	myConfig := ConfigFile{
		ServerConfig: ServerConfig{
			ListenAddress: "127.0.0.1:8080",
		},
	}
	if err := json.Unmarshal(data, &myConfig); err != nil {
		return ConfigFile{}, err
	}

	return myConfig, nil
}
