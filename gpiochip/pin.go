package gpiochip

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Pin struct {
	chip         int
	offset       int
	fd           int
	flags        uint32
	lastEdgeTime time.Time
	debounce     time.Duration
}

func (p *Pin) SetInput() error {
	return p.twiddleFlags(GPIOHANDLE_REQUEST_INPUT, GPIOHANDLE_REQUEST_OUTPUT)
}

func (p *Pin) SetOutput() error {
	return p.twiddleFlags(GPIOHANDLE_REQUEST_OUTPUT, GPIOHANDLE_REQUEST_INPUT)
}

func (p *Pin) SetActiveLow() error {
	return p.twiddleFlags(GPIOHANDLE_REQUEST_ACTIVE_LOW, 0)
}

const minDebounceDuration = 20 * time.Millisecond

func (p *Pin) SetDebounce(d time.Duration) error {
	if d < minDebounceDuration {
		return fmt.Errorf("Debounce duration too short (%v<%v)", d, minDebounceDuration)
	}

	p.debounce = d
	return nil
}

func (p *Pin) WriteBool(v bool) error {
	value := uint8(0)
	if v {
		value = 1
	}

	return WriteLine(p.fd, value)
}

func (p *Pin) ReadBool() (bool, error) {
	v, err := ReadLine(p.fd)
	if err != nil {
		return false, err
	}

	if v != 0 {
		return true, nil
	}
	return false, nil
}

func (p *Pin) GetEpollEvent(onRising, onFalling bool) (*syscall.EpollEvent, error) {
	cfd, err := getFdForController(p.chip)
	if err != nil {
		return nil, err
	}

	events := uint32(0)
	if onRising {
		events |= GPIOEVENT_REQUEST_RISING_EDGE
	}
	if onFalling {
		events |= GPIOEVENT_REQUEST_FALLING_EDGE
	}

	if p.fd >= 0 {
		syscall.Close(p.fd)
	}

	p.fd, err = GetLineEventFd(cfd, p.offset, p.flags, events)
	if err != nil {
		return nil, err
	}

	return &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(p.fd)}, nil
}

func (p *Pin) IdentifyEdge(ev *syscall.EpollEvent) (r, f bool) {
	ts_ns, edge, err := ReadEvent(p.fd)
	if err != nil {
		return false, false
	}

	ts := time.Unix(0, int64(ts_ns))
	if ts.Sub(p.lastEdgeTime) < p.debounce {
		return false, false
	}

	p.lastEdgeTime = ts

	if (edge & GPIOEVENT_EVENT_RISING_EDGE) != 0 {
		return true, false
	} else if (edge & GPIOEVENT_EVENT_FALLING_EDGE) != 0 {
		return false, true
	}

	return false, false
}

func (p *Pin) twiddleFlags(set, clear uint32) error {
	cfd, err := getFdForController(p.chip)
	if err != nil {
		return err
	}

	p.flags = p.flags &^ clear
	p.flags = p.flags | set

	if p.fd >= 0 {
		syscall.Close(p.fd)
	}

	p.fd, err = GetLineFd(cfd, p.offset, p.flags)
	if err != nil {
		return err
	}

	return nil
}

var controllerMap = make(map[int]int)

func getFdForController(chip int) (int, error) {
	if fd, ok := controllerMap[chip]; ok {
		return fd, nil
	}

	dev := fmt.Sprintf("/dev/gpiochip%d", chip)
	cfd, err := syscall.Open(dev, syscall.O_RDONLY, 0)
	if err != nil {
		return -1, fmt.Errorf("Can't open %v: %v", dev, err)
	}

	// Cache the controller fd
	controllerMap[chip] = cfd

	return cfd, nil
}

const pinPrefix = "gpiochip"

func RecognisePin(name string) bool {
	return strings.HasPrefix(name, pinPrefix)
}

func CreatePin(name string) (*Pin, error) {
	if strings.HasPrefix(name, pinPrefix) {
		parts := strings.Split(strings.TrimPrefix(name, pinPrefix), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Can't parse pin number: %v", name)
		}

		chip, err := strconv.ParseUint(parts[0], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("Can't parse pin chip number: %v", name)
		}
		offset, err := strconv.ParseUint(parts[1], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("Can't parse pin offset number: %v", name)
		}

		cfd, err := getFdForController(int(chip))
		if err != nil {
			return nil, fmt.Errorf("Can't get fd for pin %v: %v", name, err)
		}

		pfd, err := GetLineFd(cfd, int(offset), 0)
		if err != nil {
			return nil, fmt.Errorf("Can't get line fd for pin %v: %v", name, err)
		}

		p := &Pin{chip: int(chip), offset: int(offset), fd: pfd, flags: 0, debounce: minDebounceDuration}
		return p, nil
	}

	return nil, fmt.Errorf("Unrecognised pin name: %v", name)
}
