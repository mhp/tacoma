package main

import (
	"encoding/json"
	"io/ioutil"
)

type ConfigFile struct {
	ServerConfig ServerConfig
	Inputs       map[string]Input
	Outputs      map[string]Output
}

type ServerConfig struct {
	ListenAddress string
}

type Input struct {
	ExportAs  string
	Invert    bool
	OnRising  string
	OnFalling string
	Method    string
	Payload   string
}

type Output struct {
	ExportAs string
	Invert   bool
	Pulse    string
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
