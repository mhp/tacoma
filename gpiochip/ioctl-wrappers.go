package gpiochip

import (
	"bytes"
	"fmt"
	"syscall"
	"unsafe"
)

var ConsumerString = "tacoma"

func GetChipInfo(fd int) (string, string, int, error) {
	res := gpiochip_info{}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), GPIO_GET_CHIPINFO_IOCTL, uintptr(unsafe.Pointer(&res))); errno != 0 {
		return "", "", 0, errno
	}

	name := string(bytes.TrimRight(res.name[:], "\000"))
	label := string(bytes.TrimRight(res.label[:], "\000"))

	return name, label, int(res.lines), nil
}

func GetLineInfo(cfd, offset int) (uint32, string, string, error) {
	r := gpioline_info{}
	r.line_offset = uint32(offset)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(cfd), GPIO_GET_LINEINFO_IOCTL, uintptr(unsafe.Pointer(&r))); errno != 0 {
		return 0, "", "", errno
	}

	name := string(bytes.TrimRight(r.name[:], "\000"))
	consumer := string(bytes.TrimRight(r.consumer[:], "\000"))

	return r.flags, name, consumer, nil
}

func GetLineFd(cfd, offset int, flags uint32) (int, error) {
	r := gpiohandle_request{}
	r.lineoffsets[0] = uint32(offset)
	r.lines = 1
	r.flags = flags
	copy(r.consumer_label[:], ConsumerString)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(cfd), GPIO_GET_LINEHANDLE_IOCTL, uintptr(unsafe.Pointer(&r))); errno != 0 {
		return -1, errno
	}

	return r.fd, nil
}

func GetLineEventFd(cfd, offset int, flags, events uint32) (int, error) {
	r := gpioevent_request{}
	r.lineoffset = uint32(offset)
	r.handleflags = flags
	r.eventflags = events
	copy(r.consumer_label[:], ConsumerString)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(cfd), GPIO_GET_LINEEVENT_IOCTL, uintptr(unsafe.Pointer(&r))); errno != 0 {
		return -1, errno
	}

	return r.fd, nil
}

func ReadEvent(fd int) (uint64, uint32, error) {
	evSize := unsafe.Sizeof(gpioevent_data{})
	buf := make([]byte, evSize)

	n, err := syscall.Read(fd, buf)
	if err != nil {
		return 0, 0, err
	}
	if n != int(evSize) {
		return 0, 0, fmt.Errorf("Incomplete event read %v, expected %v bytes", n, evSize)
	}

	r := (*gpioevent_data)(unsafe.Pointer(&buf[0]))

	return r.timestamp, r.id, nil
}

func ReadLine(fd int) (uint8, error) {
	r := gpiohandle_data{}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), GPIOHANDLE_GET_LINE_VALUES_IOCTL, uintptr(unsafe.Pointer(&r))); errno != 0 {
		return 0, errno
	}
	return r.values[0], nil
}

func WriteLine(fd int, v uint8) error {
	r := gpiohandle_data{}
	r.values[0] = v
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), GPIOHANDLE_SET_LINE_VALUES_IOCTL, uintptr(unsafe.Pointer(&r))); errno != 0 {
		return errno
	}
	return nil
}

// _IOR is a helper function used by gpio.h.go
func _IOR(typ, nr, size uintptr) uintptr {
	return uintptr((2 << 30) | (typ << 8) | (nr) | (size << 16))
}

// _IOWR is a helper function used by gpio.h.go
func _IOWR(typ, nr, size uintptr) uintptr {
	return uintptr((3 << 30) | (typ << 8) | (nr) | (size << 16))
}
