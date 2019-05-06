package ads1015

import (
	"syscall"
)

// See: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/Documentation/i2c/dev-interface

const (
	I2C_SLAVE = 0x0703
)

func SetSlaveAddress(fd, address int) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), I2C_SLAVE, uintptr(address)); errno != 0 {
		return errno
	}

	return nil
}
