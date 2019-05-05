package main

import (
	"html/template"
	"net/http"
	"path"
	"sort"
)

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
	if export := p.Endpoint(); export != "" && p.Exported() {
		http.Handle(path.Clean("/"+export), p)
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
	    <td>{{.Endpoint}}{{if not .Exported}}<i> (Unexported)</i>{{end}}</td>
	    <td>{{if .Inverted}}!{{end}}{{.PinName}}</td>
	    <td>{{.Direction}}</td>
	    <td>{{.String}}</td>
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
