package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// PinHandler is the generic interface to a pin, for the purposes
// of the http UI, and also during template evaluation for trigger
// body creation.
type PinHandler interface {
	Endpoint() string
	PinName() string
	Exported() bool
	Direction() string
	Inverted() bool
	String() string

	ServeHTTP(http.ResponseWriter, *http.Request)
}

// readablePin is a minimal interface that all pins implement if they
// are able to be used with the http UI and trigger templates
type readablePin interface {
	Read() (string, error)
}

type pinHandler struct {
	name     string
	endpoint string
	exported bool
	inverted bool
	input    bool
	pin      readablePin
}

func newInputPinHandler(name string, pin GenericInputPin, cfg Input) PinHandler {
	return pinHandler{name: cfg.Pin,
		endpoint: name,
		exported: !cfg.Hidden,
		inverted: cfg.Invert,
		input:    true,
		pin:      pin,
	}
}

func newOutputPinHandler(name string, pin GenericOutputPin, cfg Output) PinHandler {
	return pinHandler{name: cfg.Pin,
		endpoint: name,
		exported: !cfg.Hidden,
		inverted: cfg.Invert,
		input:    false,
		pin:      pin,
	}
}

func (h pinHandler) PinName() string {
	return h.name
}

func (h pinHandler) Endpoint() string {
	return h.endpoint
}

func (h pinHandler) Exported() bool {
	return h.exported
}

func (h pinHandler) Inverted() bool {
	return h.inverted
}

func (h pinHandler) Direction() string {
	if h.input {
		return "input"
	}
	return "output"
}

// String lets a PinHandler have the underlying value read whilst
// evaluating a template for a trigger body
func (h pinHandler) String() string {
	v, err := h.pin.Read()
	if err != nil {
		return "?"
	}
	return v
}

// ServeHTTP lets a PinHandler be registered with an HTTP server and handle
// GET/PUT requests for the underlying pin
func (h pinHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		v, err := h.pin.Read()
		if err != nil {
			http.Error(w, "Unable to read value", http.StatusInternalServerError)
		} else {
			fmt.Fprintf(w, "%v", v)
		}
	} else if r.Method == "PUT" && !h.input {
		if bbody, err := ioutil.ReadAll(r.Body); err != nil {
			http.Error(w, "Unable to read PUT body", http.StatusInternalServerError)
			return
		} else {
			body := string(bbody)

			// Since h.input is false, we expect this to always succeed
			op, ok := h.pin.(GenericOutputPin)
			if !ok {
				http.Error(w, "Unable to get output pin", http.StatusInternalServerError)
				return
			}

			if err := op.Write(body); err != nil {
				http.Error(w, "Unable to write value", http.StatusInternalServerError)
				return
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
