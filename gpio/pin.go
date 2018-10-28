package gpio

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Pin struct {
	id       uint
	basePath string
}

const (
	gpioExport   = "/sys/class/gpio/export"
	gpioUnexport = "/sys/class/gpio/unexport"
)

func dirExists(f string) bool {
	fi, err := os.Stat(f)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func (p *Pin) Export(id uint) error {
	p.id = id
	p.basePath = fmt.Sprintf("/sys/class/gpio/gpio%d/", id)

	if dirExists(p.basePath) {
		return nil
	}

	if err := ioutil.WriteFile(gpioExport, []byte(fmt.Sprintf("%d", p.id)), 0222); err != nil {
		return err
	}

	// This is a hack, to wait until we can access the pin we've just exported
	// udev scripts take time to run, so keep polling for access until we get it
	// or give up
	for i := 0; i < 10; i++ {
		if err := syscall.Access(p.basePath, 7 /* R_OK, W_OK, X_OK */); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("Can't access exported directory for pin %v", p.id)
}

func (p Pin) Unexport() error {
	if !dirExists(p.basePath) {
		return nil
	}

	if err := ioutil.WriteFile(gpioUnexport, []byte(fmt.Sprintf("%d", p.id)), 0222); err != nil {
		return err
	}

	if dirExists(p.basePath) {
		return fmt.Errorf("Can't unexport pin %v", p.id)
	}

	return nil
}

func (p Pin) writeFile(fn, value string) error {
	data := []byte(value)
	return ioutil.WriteFile(p.basePath+fn, data, 0220)
}

func (p Pin) readFile(fn string) (string, error) {
	v, err := ioutil.ReadFile(p.basePath + fn)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(v)), nil
}

func (p Pin) SetInput() error {
	return p.writeFile("direction", "in")
}

func (p Pin) SetOutput() error {
	return p.writeFile("direction", "out")
}

func (p Pin) IsInput() (bool, error) {
	v, err := p.readFile("direction")
	return v == "in", err
}

func (p Pin) SetActiveLow() error {
	return p.writeFile("active_low", "1")
}

func (p Pin) SetActiveHigh() error {
	return p.writeFile("active_low", "0")
}

func (p Pin) IsActiveLow() (bool, error) {
	v, err := p.readFile("active_low")
	return v == "1", err
}

func (p Pin) Write(v bool) error {
	if v {
		return p.writeFile("value", "1")
	}
	return p.writeFile("value", "0")
}

func (p Pin) Read() (string, error) {
	return p.readFile("value")
}

func (p Pin) GetEpollEvent(onRising, onFalling bool) (*syscall.EpollEvent, error) {
	edge := "none"
	switch {
	case onRising && !onFalling:
		edge = "rising"
	case !onRising && onFalling:
		edge = "falling"
	case onRising && onFalling:
		edge = "both"
	default:
		return nil, fmt.Errorf("unexpected edge combination - neither")
	}

	if err := p.writeFile("edge", edge); err != nil {
		return nil, err
	}

	// Use syscall.Open instead of os.Open to ensure our fd doesn't get garbage-collected
	fd, err := syscall.Open(p.basePath + "value", syscall.O_RDONLY | syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}

	return &syscall.EpollEvent{Events: -syscall.EPOLLET | syscall.EPOLLPRI, Fd: int32(fd)}, nil
}

func (p Pin) IdentifyEdge(ev *syscall.EpollEvent) (r, f bool) {
	v, err := p.Read()
	fmt.Println("Got events: ", ev.Events, "currently", v, err)
	return true, false
}


func RecognisePin(name string) bool {
	return strings.HasPrefix(name, "gpio")
}

func CreatePin(name string) (*Pin, error) {
	if strings.HasPrefix(name, "gpio") {
		num, err := strconv.ParseUint(name[4:], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("Can't parse pin number (%v): %v", name, err)
		}
		p := &Pin{}
		return p, p.Export(uint(num))
	}

	return nil, fmt.Errorf("Unrecognised pin name: %v", name)
}
