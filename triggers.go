package main

import (
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"text/template"
)

const DefaultMethod = "PUT"
const DefaultTemplate = "{{if .RisingEdge}}1{{else}}0{{end}}"

var Client http.Client

type triggerInfo struct {
	p         TriggeringPin
	onRising  string
	onFalling string
	method    string
	tpl       *template.Template
}

func (ti *triggerInfo) send(rising bool, url string) error {
	var body strings.Builder

	err := ti.tpl.Execute(&body, struct {
		RisingEdge  bool
		FallingEdge bool
		Pin         PinMap
	}{
		RisingEdge:  rising,
		FallingEdge: !rising,
		Pin:         pinMap,
	})

	if err != nil {
		return fmt.Errorf("Template execution failed: %v", err)
	}

	req, err := http.NewRequest(ti.method, url, strings.NewReader(body.String()))
	if err != nil {
		return fmt.Errorf("Can't create HTTP request: %v", err)
	}

	resp, err := Client.Do(req)
	if err != nil {
		return fmt.Errorf("Can't do HTTP request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed: %v", resp.Status)
	}

	return nil
}

func (ti *triggerInfo) SendRising() error {
	if ti.onRising == "" {
		return nil
	}

	return ti.send(true, ti.onRising)
}

func (ti *triggerInfo) SendFalling() error {
	if ti.onFalling == "" {
		return nil
	}

	return ti.send(false, ti.onFalling)
}

type Triggers struct {
	epollFd int
	pins    map[int]triggerInfo
}

func NewTriggers() (*Triggers, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	return &Triggers{fd, make(map[int]triggerInfo)}, nil
}

func (t *Triggers) Add(p TriggeringPin, onRising string, onFalling string, method string, payload string) error {
	if method == "" {
		method = DefaultMethod
	}

	if payload == "" {
		payload = DefaultTemplate
	}

	tpl, err := template.New("trigger").Parse(payload)
	if err != nil {
		return fmt.Errorf("cannot parse payload template: %v", err)
	}

	ev, err := p.GetEpollEvent(onRising != "", onFalling != "")
	if err != nil {
		return fmt.Errorf("cannot configure pin: %v", err)
	}

	err = syscall.EpollCtl(t.epollFd, syscall.EPOLL_CTL_ADD, int(ev.Fd), ev)
	if err != nil {
		return fmt.Errorf("epoll: %v", err)
	}

	t.pins[int(ev.Fd)] = triggerInfo{p, onRising, onFalling, method, tpl}

	return nil
}

// pinMap is a map of pin value functions indexed by pin name
// It can be used when evaluating templates so multiple pins
// can be sampled as part of the same event.
type PinMap map[string]fmt.Stringer

var pinMap PinMap

// AddContext adds the pin to the context used during template evaluation
func (*Triggers) AddContext(p PinHandler) {
	if pinMap == nil {
		pinMap = make(PinMap)
	}

	pinMap[p.PinName()] = p
}

func (t *Triggers) Wait() {
	events := make([]syscall.EpollEvent, 1)

	for {
		n, err := syscall.EpollWait(t.epollFd, events, -1)
		if err != nil && err != syscall.EINTR {
			fmt.Println("epoll_wait returned error:", err)
			return
		}

		if n > 0 {
			fd := int(events[0].Fd)

			if ti, ok := t.pins[fd]; ok {
				r, f := ti.p.IdentifyEdge(&events[0])

				if r {
					ti.SendRising()
				}

				if f {
					ti.SendFalling()
				}
			} else {
				fmt.Println("epoll returned event for unrecognised fd", fd)
			}
		}
	}
}
