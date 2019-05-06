package main

import (
	"strconv"
)

type wrappedAI struct {
	AnalogueInputPin
}

func WrapAnalogueInput(p AnalogueInputPin) GenericInputPin {
	return &wrappedAI{p}
}

func (p *wrappedAI) Read() (string, error) {
	v, err := p.ReadValue()
	if err != nil {
		return "n/a", nil
	}

	return strconv.Itoa(v), nil
}
