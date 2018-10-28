package gpiochip

import (
	"unsafe"
)

// Constants and structures copied from include/uapi/linux/gpio.h of kernel version 4.14.76
// and then manually converted to something sufficiently go-like

/* SPDX-License-Identifier: GPL-2.0 WITH Linux-syscall-note */
/*
 * <linux/gpio.h> - userspace ABI for the GPIO character devices
 *
 * Copyright (C) 2016 Linus Walleij
 *
 * This program is free software; you can redistribute it and/or modify it
 * under the terms of the GNU General Public License version 2 as published by
 * the Free Software Foundation.
 */

/**
 * struct gpiochip_info - Information about a certain GPIO chip
 * @name: the Linux kernel name of this GPIO chip
 * @label: a functional name for this GPIO chip, such as a product
 * number, may be NULL
 * @lines: number of GPIO lines on this chip
 */
type gpiochip_info struct {
	name  [32]byte
	label [32]byte
	lines uint32
}

/* Informational flags */
const (
	GPIOLINE_FLAG_KERNEL      = (1 << 0) /* Line used by the kernel */
	GPIOLINE_FLAG_IS_OUT      = (1 << 1)
	GPIOLINE_FLAG_ACTIVE_LOW  = (1 << 2)
	GPIOLINE_FLAG_OPEN_DRAIN  = (1 << 3)
	GPIOLINE_FLAG_OPEN_SOURCE = (1 << 4)
)

/**
 * struct gpioline_info - Information about a certain GPIO line
 * @line_offset: the local offset on this GPIO device, fill this in when
 * requesting the line information from the kernel
 * @flags: various flags for this line
 * @name: the name of this GPIO line, such as the output pin of the line on the
 * chip, a rail or a pin header name on a board, as specified by the gpio
 * chip, may be NULL
 * @consumer: a functional name for the consumer of this GPIO line as set by
 * whatever is using it, will be NULL if there is no current user but may
 * also be NULL if the consumer doesn't set this up
 */
type gpioline_info struct {
	line_offset uint32
	flags       uint32
	name        [32]byte
	consumer    [32]byte
}

/* Maximum number of requested handles */
const GPIOHANDLES_MAX = 64

/* Linerequest flags */
const (
	GPIOHANDLE_REQUEST_INPUT       = (1 << 0)
	GPIOHANDLE_REQUEST_OUTPUT      = (1 << 1)
	GPIOHANDLE_REQUEST_ACTIVE_LOW  = (1 << 2)
	GPIOHANDLE_REQUEST_OPEN_DRAIN  = (1 << 3)
	GPIOHANDLE_REQUEST_OPEN_SOURCE = (1 << 4)
)

/**
 * struct gpiohandle_request - Information about a GPIO handle request
 * @lineoffsets: an array desired lines, specified by offset index for the
 * associated GPIO device
 * @flags: desired flags for the desired GPIO lines, such as
 * GPIOHANDLE_REQUEST_OUTPUT, GPIOHANDLE_REQUEST_ACTIVE_LOW etc, OR:ed
 * together. Note that even if multiple lines are requested, the same flags
 * must be applicable to all of them, if you want lines with individual
 * flags set, request them one by one. It is possible to select
 * a batch of input or output lines, but they must all have the same
 * characteristics, i.e. all inputs or all outputs, all active low etc
 * @default_values: if the GPIOHANDLE_REQUEST_OUTPUT is set for a requested
 * line, this specifies the default output value, should be 0 (low) or
 * 1 (high), anything else than 0 or 1 will be interpreted as 1 (high)
 * @consumer_label: a desired consumer label for the selected GPIO line(s)
 * such as "my-bitbanged-relay"
 * @lines: number of lines requested in this request, i.e. the number of
 * valid fields in the above arrays, set to 1 to request a single line
 * @fd: if successful this field will contain a valid anonymous file handle
 * after a GPIO_GET_LINEHANDLE_IOCTL operation, zero or negative value
 * means error
 */
type gpiohandle_request struct {
	lineoffsets    [GPIOHANDLES_MAX]uint32
	flags          uint32
	default_values [GPIOHANDLES_MAX]uint8
	consumer_label [32]byte
	lines          uint32
	fd             int
}

/**
 * struct gpiohandle_data - Information of values on a GPIO handle
 * @values: when getting the state of lines this contains the current
 * state of a line, when setting the state of lines these should contain
 * the desired target state
 */
type gpiohandle_data struct {
	values [GPIOHANDLES_MAX]uint8
}

var (
	GPIOHANDLE_GET_LINE_VALUES_IOCTL = _IOWR(0xB4, 0x08, unsafe.Sizeof(gpiohandle_data{}))
	GPIOHANDLE_SET_LINE_VALUES_IOCTL = _IOWR(0xB4, 0x09, unsafe.Sizeof(gpiohandle_data{}))
)

/* Eventrequest flags */
const (
	GPIOEVENT_REQUEST_RISING_EDGE  = (1 << 0)
	GPIOEVENT_REQUEST_FALLING_EDGE = (1 << 1)
	GPIOEVENT_REQUEST_BOTH_EDGES   = ((1 << 0) | (1 << 1))
)

/**
 * struct gpioevent_request - Information about a GPIO event request
 * @lineoffset: the desired line to subscribe to events from, specified by
 * offset index for the associated GPIO device
 * @handleflags: desired handle flags for the desired GPIO line, such as
 * GPIOHANDLE_REQUEST_ACTIVE_LOW or GPIOHANDLE_REQUEST_OPEN_DRAIN
 * @eventflags: desired flags for the desired GPIO event line, such as
 * GPIOEVENT_REQUEST_RISING_EDGE or GPIOEVENT_REQUEST_FALLING_EDGE
 * @consumer_label: a desired consumer label for the selected GPIO line(s)
 * such as "my-listener"
 * @fd: if successful this field will contain a valid anonymous file handle
 * after a GPIO_GET_LINEEVENT_IOCTL operation, zero or negative value
 * means error
 */
type gpioevent_request struct {
	lineoffset     uint32
	handleflags    uint32
	eventflags     uint32
	consumer_label [32]byte
	fd             int
}

/**
 * GPIO event types
 */
const (
	GPIOEVENT_EVENT_RISING_EDGE  = 0x01
	GPIOEVENT_EVENT_FALLING_EDGE = 0x02
)

/**
 * struct gpioevent_data - The actual event being pushed to userspace
 * @timestamp: best estimate of time of event occurrence, in nanoseconds
 * @id: event identifier
 */
type gpioevent_data struct {
	timestamp uint64
	id        uint32
	__pad     uint32 // C and go padding rules differ, so fudge this!
}

var (
	GPIO_GET_CHIPINFO_IOCTL   = _IOR(0xB4, 0x01, unsafe.Sizeof(gpiochip_info{}))
	GPIO_GET_LINEINFO_IOCTL   = _IOWR(0xB4, 0x02, unsafe.Sizeof(gpioline_info{}))
	GPIO_GET_LINEHANDLE_IOCTL = _IOWR(0xB4, 0x03, unsafe.Sizeof(gpiohandle_request{}))
	GPIO_GET_LINEEVENT_IOCTL  = _IOWR(0xB4, 0x04, unsafe.Sizeof(gpioevent_request{}))
)
