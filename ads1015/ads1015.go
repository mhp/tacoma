package ads1015

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// See: http://www.ti.com/lit/ds/symlink/ads1015.pdf

type adc struct {
	fd   int
	lock sync.Mutex
}

func (adc *adc) Convert(ch int) (int, error) {
	adc.lock.Lock()
	defer adc.lock.Unlock()

	config := []byte{
		0x01,	// Write to config reigster
		0xC3 | ((byte(ch)<<4)&0x30),	// Start single ended conversion, 4.096V FSR
		0x83,	// 1600sps, alert/rdy --> hi-Z
	}

	n, err := syscall.Write(adc.fd, config)
	if err != nil {
		return 0, err
	} else if n != len(config) {
		return 0, errors.New("bad len (cfg)")
	}

	// 1600 samples per second implies a delay of 625us
	time.Sleep(625 * time.Microsecond)

	// Prepare to read - write the address register first
	setConvReg := []byte{
		0x00,	// Select conversion register
	}
	n, err = syscall.Write(adc.fd, setConvReg)
	if err != nil {
		return 0, err
	} else if n != len(setConvReg) {
		return 0, errors.New("bad len (select)")
	}

	result := make([]byte, 2)
	n, err = syscall.Read(adc.fd, result)
	if err != nil {
		return 0, err
	} else if n != len(result) {
		return 0, errors.New("bad len (read)")
	}

	// Result is formatted MSB first, with bottom 4 bits as zero
	// Repack it it make a 12-bit value
	return int(result[0])<<4 | int(result[1]>>4), nil
}

type Channel struct {
	Name    string
	channel int
	adc     *adc
}

func (c *Channel) SetInput() error {
	// Converter pins are always inputs - no action required
	return nil
}

// 12-bit ADC
func (c *Channel) MaxValue() int {
	return 4095
}

// Single-ended so never below zero
func (c *Channel) MinValue() int {
	return 0
}

func (c *Channel) ReadValue() (int, error) {
	return c.adc.Convert(c.channel)
}

var adcMap = make(map[int]*adc)

func getADC(dev string, addr int) (*adc, error) {
	// FIXME This assumes only one bus, so
	// a single cache indexed by address is adequate
	if adc, ok := adcMap[addr]; ok {
		return adc, nil
	}

	fd, err := syscall.Open(dev, syscall.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("Can't open %v: %v", dev, err)
	}

	if err = SetSlaveAddress(fd, addr); err != nil {
		return nil, fmt.Errorf("Can't configure I2C slave address %v: %v", addr, err)
	}

	// Make an instance of the analogue to digital converter
	adc := &adc{fd: fd}

	// Cache the controller fd
	adcMap[addr] = adc

	return adc, nil
}

const defaultAddress = 0x48
const defaultBus = "/dev/i2c-1"

var adsNames = regexp.MustCompile(`\A` +
	// Base name followed by channel number
	`ads1015:([0123])` +
	// Optional i2c address in hex ("@48") starting with '@'
	`(?:@([[:xdigit:]]{2}))?` +
	`\z`)

const (
	submatchAll = iota
	submatchChan
	submatchAddr
	numSubmatchesExpected
)

func RecognisePin(name string) bool {
	return adsNames.MatchString(name)
}

func CreatePin(name string) (*Channel, error) {
	submatches := adsNames.FindStringSubmatch(name)
	if len(submatches) != numSubmatchesExpected {
		return nil, fmt.Errorf("Can't parse pin name: %v", name)
	}

	chNum, err := strconv.ParseUint(submatches[submatchChan], 10, 8)
	if err != nil {
		return nil, fmt.Errorf("Can't parse channel number: %v", name)
	}

	thisAddr := defaultAddress
	if len(submatches[submatchAddr]) > 0 {
		addr, err := strconv.ParseUint(submatches[submatchAddr], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("Can't parse device address: %v", name)
		}
		thisAddr = int(addr)
	}

	if c, err := getADC(defaultBus, thisAddr); err != nil {
		return nil, err
	} else {
		return &Channel{name, int(chNum), c}, nil
	}
}
