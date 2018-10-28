package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mhp/tacoma/fakeio"
	"github.com/mhp/tacoma/gpio"
	"github.com/mhp/tacoma/gpiochip"
)

func main() {
	if len(os.Args) > 2 {
		fmt.Println("Usage:", os.Args[0], "[config.json]")
		os.Exit(1)
	}

	cfgFile := "config.json"
	if len(os.Args) == 2 {
		cfgFile = os.Args[1]
	}

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}

	myHandlers := Handlers{cfg.ServerConfig, nil}
	myTriggers, err := NewTriggers()
	if err != nil {
		fmt.Println("Error initialising epoll:", err)
		os.Exit(1)
	}

	// iterate over outputs, enabling pins and adding handlers
	for name, op := range cfg.Outputs {
		p, err := getPin(name)
		if err != nil {
			fmt.Println("Bad output", name, err)
			os.Exit(1)
		}
		if err = p.SetOutput(); err != nil {
			fmt.Println("Bad output (can't set as output)", name, err)
			os.Exit(1)
		}
		if op.Invert {
			if err = p.SetActiveLow(); err != nil {
				fmt.Println("Bad output (can't set active low)", name, err)
				os.Exit(1)
			}
		}
		myHandlers.Add(OutputHandler{Name: name, Pin: p, Cfg: op})
	}

	// iterate over inputs, enabling pins, adding handlers, setting up triggers
	for name, ip := range cfg.Inputs {
		p, err := getPin(name)
		if err != nil {
			fmt.Println("Bad input", name, err)
			os.Exit(1)
		}
		if err = p.SetInput(); err != nil {
			fmt.Println("Bad input (can't set as input)", name, err)
			os.Exit(1)
		}
		if ip.Invert {
			if err = p.SetActiveLow(); err != nil {
				fmt.Println("Bad input (can't set active low)", name, err)
				os.Exit(1)
			}
		}
		if ip.Method == "" {
			ip.Method = "PUT"
		}

		if ip.Payload == "" {
			ip.Payload = "{{if .RisingEdge}}1{{else}}0{{end}}"
		}
		myHandlers.Add(InputHandler{Name: name, Pin: p, Cfg: ip})
		if err := myTriggers.Add(p, ip.OnRising, ip.OnFalling, ip.Method, ip.Payload); err != nil {
			fmt.Println("Bad input", name, err)
			os.Exit(1)
		}
	}

	go myTriggers.Wait()

	http.Handle("/", myHandlers)

	if err := http.ListenAndServe(cfg.ServerConfig.ListenAddress, nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type Pin interface {
	SetInput() error
	SetOutput() error
	SetActiveLow() error

	Write(bool) error
	Read() (string, error)
}

func getPin(name string) (Pin, error) {
	if gpiochip.RecognisePin(name) {
		return gpiochip.CreatePin(name)
	}

	if gpio.RecognisePin(name) {
		return gpio.CreatePin(name)
	}

	if fakeio.RecognisePin(name) {
		return fakeio.CreatePin(name)
	}

	return nil, fmt.Errorf("Unknown pin type: %v", name)
}
