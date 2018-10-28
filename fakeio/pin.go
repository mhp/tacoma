package fakeio

import (
	"fmt"
	"strings"
	"syscall"
	"time"
)

type Pin struct {
	Name string
	high bool
}

func (p *Pin) SetInput() error {
	fmt.Println(p.Name, "direction --> IN")
	return nil
}

func (p *Pin) SetOutput() error {
	fmt.Println(p.Name, "direction --> OUT")
	return nil
}

func (p *Pin) SetActiveLow() error {
	fmt.Println(p.Name, "set active low")
	return nil
}

func (p *Pin) Write(v bool) error {
	fmt.Println(p.Name, "set output", v)
	p.high = v
	return nil
}

func (p *Pin) Read() (string, error) {
	fmt.Println(p.Name, "reading", p.high)

	if p.high {
		return "1", nil
	}

	return "0", nil
}

func (p *Pin) GetEpollEvent(r, f bool) (*syscall.EpollEvent, error) {
	pipes := make([]int, 2)

	if err := syscall.Pipe2(pipes, syscall.O_CLOEXEC); err != nil {
		return nil, err
	}

	go func(fd int) {
		t := time.NewTicker(2 * time.Second)
		buf := make([]byte, 1)

		for _ = range t.C {
			if _, err := syscall.Write(fd, buf); err != nil {
				fmt.Println("Can't write from ticker:", p.Name, err)
			}
		}
	}(pipes[1])

	return &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(pipes[0])}, nil
}

func (p *Pin) IdentifyEdge(e *syscall.EpollEvent) (r, f bool) {
	// First, read from the pipe to drain it
	buf := make([]byte, 1)
	syscall.Read(int(e.Fd), buf)

	p.high = !p.high

	if p.high {
		return true, false
	}

	return false, true
}

func RecognisePin(name string) bool {
	return strings.HasPrefix(name, "fakeio")
}

func CreatePin(name string) (*Pin, error) {
	return &Pin{name, false}, nil
}
