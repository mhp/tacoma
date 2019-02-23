package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"sort"
)

type ReadablePin interface {
	Read() (string, error)
}

type pinHandler struct {
	name     string
	endpoint string
	inverted bool
	input    bool

	Pin ReadablePin
}

func newInputPinHandler(name string, pin GenericInputPin, cfg Input) PinHandler {
	return pinHandler{name: name,
		endpoint: cfg.ExportAs,
		inverted: cfg.Invert,
		input:    true,
		Pin:      pin,
	}
}

func newOutputPinHandler(name string, pin GenericOutputPin, cfg Output) PinHandler {
	return pinHandler{name: name,
		endpoint: cfg.ExportAs,
		inverted: cfg.Invert,
		input:    false,
		Pin:      pin,
	}
}

func (h pinHandler) PinName() string {
	return h.name
}

func (h pinHandler) Endpoint() string {
	return h.endpoint
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

func (h pinHandler) Value() string {
	v, err := h.Pin.Read()
	if err != nil {
		return "?"
	}
	return v
}

func (h pinHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		v, err := h.Pin.Read()
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
			op, ok := h.Pin.(GenericOutputPin)
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

type PinHandler interface {
	Endpoint() string
	PinName() string
	Direction() string
	Inverted() bool
	Value() string

	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Make []PinHandler sortable by endpoint
type ByEndpoint []PinHandler

func (p ByEndpoint) Len() int           { return len(p) }
func (p ByEndpoint) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByEndpoint) Less(i, j int) bool { return p[i].Endpoint() < p[j].Endpoint() }

type Handlers struct {
	Cfg  ServerConfig
	Pins []PinHandler
}

func (hs *Handlers) Add(p PinHandler) {
	hs.Pins = append(hs.Pins, p)
	if export := p.Endpoint(); export != "" {
		http.Handle(export, p)
	}
}

func (hs Handlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("status").Parse(`<!DOCTYPE html>
	<html><head>
	<title>Tacoma</title>
	<style>
	h1 {text-align: center}
	table {width: 80%; margin: auto}
	th {text-align: left; background: #D0D0D0}     
	td, th {padding: 0.2em}
    	tr:nth-child(even) td {background: #F0F0F0}           
    	tr:nth-child(odd) td {background: #FDFDFD}
	</style>
	<meta http-equiv="refresh" content="30">
	</head><body>
	<h1>Tacoma on {{.Cfg.ListenAddress}}</h1>
	<table>
	<thead><tr><th>Endpoint</th><th>Pin</th><th>Direction</th><th>Value</th></tr></thead>
	<tbody>{{ range .P }}
	  <tr>
	    <td>{{if gt (len .Endpoint) 0 }}{{.Endpoint}}{{else}}<i>Unexported</i>{{end}}</td>
	    <td>{{if .Inverted}}!{{end}}{{.PinName}}</td>
	    <td>{{.Direction}}</td>
	    <td>{{.Value}}</td>
	  </tr>
	{{ end }}</tbody>
	</table>
	</body></html>
	`)

	if err != nil {
		http.Error(w, "500 Internal server fault", 500)
		return
	}

	sort.Stable(ByEndpoint(hs.Pins))

	err = t.Execute(w, struct {
		Cfg ServerConfig
		P   []PinHandler
	}{hs.Cfg, hs.Pins})
	if err != nil {
		http.Error(w, "500 Internal server fault", 500)
	}
}
