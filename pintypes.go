package main

import (
	"syscall"
)

type Pin interface {
}

type DigitalPin interface {
	SetActiveLow() error
}

type InputPin interface {
	SetInput() error
	Read() (string, error)
}

type OutputPin interface {
	SetOutput() error
	Read() (string, error)
	Write(bool) error
}

type TriggeringPin interface {
	GetEpollEvent(onRising, onFalling bool) (*syscall.EpollEvent, error)
	IdentifyEdge(*syscall.EpollEvent) (rising, falling bool)
}
