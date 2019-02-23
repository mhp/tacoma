package main

import (
	"strings"
)

type wrappedDI struct {
	DigitalInputPin
}

func WrapDigitalInput(p DigitalInputPin) GenericInputPin {
	return &wrappedDI{p}
}

func (p *wrappedDI) Read() (string, error) {
	v, err := p.ReadBool()
	if err != nil {
		return "n/a", nil
	}

	if v {
		return "1", nil
	}
	return "0", nil
}

type wrappedDO struct {
	DigitalOutputPin
}

func WrapDigitalOutput(p DigitalOutputPin) GenericOutputPin {
	return &wrappedDO{p}
}

func (p *wrappedDO) Write(value string) error {
	var v bool

	switch {
	case strings.HasPrefix(value, "false"),
		strings.HasPrefix(value, "low"),
		strings.HasPrefix(value, "0"):
		v = false
	default:
		v = true
	}

	return p.WriteBool(v)
}

func (p *wrappedDO) Read() (string, error) {
	v, err := p.ReadBool()
	if err != nil {
		return "n/a", nil
	}

	if v {
		return "1", nil
	}
	return "0", nil
}
