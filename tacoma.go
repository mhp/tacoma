package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mhp/tacoma/fakeio"
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
	for name, cfg := range cfg.Outputs {
		p, err := getPin(name)
		if err != nil {
			fmt.Println("Bad output", name, err)
			os.Exit(1)
		}

		op, ok := p.(OutputPin)
		if !ok {
			fmt.Println("Pin can't be used as an output", name)
			os.Exit(1)

		} else if err = op.SetOutput(); err != nil {
			fmt.Println("Bad output (can't set as output)", name, err)
			os.Exit(1)
		}

		if cfg.Invert {
			if dp, ok := p.(DigitalOutputPin); !ok {
				fmt.Println("Pin doesn't support inverted operation", name)
				os.Exit(1)
			} else if err = dp.SetActiveLow(); err != nil {
				fmt.Println("Bad output (can't set active low)", name, err)
				os.Exit(1)
			}
		}

		var ph PinHandler = nil
		switch pin := p.(type) {
		case GenericOutputPin:
			ph = newOutputPinHandler(name, pin, cfg)
		case DigitalOutputPin:
			ph = newOutputPinHandler(name, WrapDigitalOutput(pin), cfg)
		default:
			fmt.Println("Can't handle pin type as output", pin)
		}

		if ph != nil {
			myHandlers.Add(ph)
			myTriggers.AddContext(ph)
		}
	}

	// iterate over inputs, enabling pins, adding handlers, setting up triggers
	for name, cfg := range cfg.Inputs {
		p, err := getPin(name)
		if err != nil {
			fmt.Println("Bad input", name, err)
			os.Exit(1)
		}

		ip, ok := p.(InputPin)
		if !ok {
			fmt.Println("Pin can't be used as an input", name)
			os.Exit(1)

		} else if err = ip.SetInput(); err != nil {
			fmt.Println("Bad input (can't set as input)", name, err)
			os.Exit(1)
		}

		if cfg.Invert {
			if dp, ok := p.(DigitalInputPin); !ok {
				fmt.Println("Pin doesn't support inverted operation", name)
				os.Exit(1)
			} else if err = dp.SetActiveLow(); err != nil {
				fmt.Println("Bad input (can't set active low)", name, err)
				os.Exit(1)
			}
		}

		if cfg.OnRising != "" || cfg.OnFalling != "" {
			if tp, ok := p.(TriggeringPin); !ok {
				fmt.Println("Pin cannot be used for event triggers", name)
				os.Exit(1)
			} else if err := myTriggers.Add(tp, cfg.OnRising, cfg.OnFalling, cfg.Method, cfg.Payload); err != nil {
				fmt.Println("Bad input", name, err)
				os.Exit(1)
			}
		}

		var ph PinHandler = nil
		switch pin := p.(type) {
		case GenericInputPin:
			ph = newInputPinHandler(name, pin, cfg)
		case DigitalInputPin:
			ph = newInputPinHandler(name, WrapDigitalInput(pin), cfg)
		default:
			fmt.Println("Can't handle pin type as input", pin)
		}

		if ph != nil {
			myHandlers.Add(ph)
			myTriggers.AddContext(ph)
		}
	}

	go myTriggers.Wait()

	http.Handle("/", myHandlers)

	if err := http.ListenAndServe(cfg.ServerConfig.ListenAddress, nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getPin(name string) (interface{}, error) {
	if gpiochip.RecognisePin(name) {
		return gpiochip.CreatePin(name)
	}

	if fakeio.RecognisePin(name) {
		return fakeio.CreatePin(name)
	}

	return nil, fmt.Errorf("Unknown pin type: %v", name)
}
