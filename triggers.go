package main

import (
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"text/template"
)

type TriggerablePin interface {
	GetEpollEvent(onRising, onFalling bool) (*syscall.EpollEvent, error)
	IdentifyEdge(*syscall.EpollEvent) (rising, falling bool)
}

var Client http.Client

type triggerInfo struct {
	p         TriggerablePin
	onRising  string
	onFalling string
	method    string
	tpl       *template.Template
}

func (ti *triggerInfo) send(rising bool, url string) error {
	var body strings.Builder

	ti.tpl.Execute(&body, struct {
		RisingEdge  bool
		FallingEdge bool
	}{
		RisingEdge:  rising,
		FallingEdge: !rising,
	})

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

func (t *Triggers) Add(p Pin, onRising string, onFalling string, method string, payload string) error {
	if onRising == "" && onFalling == "" {
		// No triggers required
		return nil
	}

	tp, ok := p.(TriggerablePin)
	if !ok {
		return fmt.Errorf("pin cannot trigger events")
	}

	tpl, err := template.New("trigger").Parse(payload)
	if err != nil {
		return fmt.Errorf("cannot parse payload template: %v", err)
	}

	ev, err := tp.GetEpollEvent(onRising != "", onFalling != "")
	if err != nil {
		return fmt.Errorf("cannot configure pin: %v", err)
	}

	err = syscall.EpollCtl(t.epollFd, syscall.EPOLL_CTL_ADD, int(ev.Fd), ev)
	if err != nil {
		return fmt.Errorf("epoll: %v", err)
	}

	t.pins[int(ev.Fd)] = triggerInfo{tp, onRising, onFalling, method, tpl}

	return nil
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
