package main

import (
	"syscall"
	"time"
)

// InputPin defines the bare minimum for a pin - it can be configured as an Input
type InputPin interface {
	SetInput() error
}

// OutputPin defines the bare minimum for a pin - it can be configured as an Output
type OutputPin interface {
	SetOutput() error
}

// DigitalInputPin defines the additional functionality of a digital input
type DigitalInputPin interface {
	SetActiveLow() error
	SetDebounce(time.Duration) error
	ReadBool() (bool, error)
}

// DigitalInputPin defines the additional functionality of a digital input
type DigitalOutputPin interface {
	SetActiveLow() error
	ReadBool() (bool, error)
	WriteBool(bool) error
}

// AnalogueInputPin defines what an analogue pin can do
type AnalogueInputPin interface {
	MinValue() int
	MaxValue() int
	ReadValue() (int, error)
}

// GenericInputPin allows a string to be read as its value
type GenericInputPin interface {
	Read() (string, error)
}

// GenericInputPin allows a string to be used as its value
type GenericOutputPin interface {
	Read() (string, error)
	Write(string) error
}

type TriggeringPin interface {
	GetEpollEvent(onRising, onFalling bool) (*syscall.EpollEvent, error)
	IdentifyEdge(*syscall.EpollEvent) (rising, falling bool)
}
