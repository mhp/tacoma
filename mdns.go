package main

import (
	"context"
	"net"
)

// mdnsAddr is the ipv4 multicast address used for lookups
var mdnsAddr = &net.UDPAddr{
	IP:   []byte{224, 0, 0, 251},
	Port: 5353,
}

// mdnsShim looks like a net.PacketConn, and transparently forwards
// all the methods to the underlying net.PacketConn.  We also implement
// the rest of the net.Conn interface
type mdnsShim struct {
	net.PacketConn
}

// Write expects to send on a connected socket, but we use a listening
// udp socket to permit responses from any host.  Therefore implement
// it by performing a WriteTo the multicast address for mdns lookups
func (s mdnsShim) Write(b []byte) (n int, err error) {
	return s.WriteTo(b, mdnsAddr)
}

// Read expects to read from a connected socket, but we use a listening
// udp socket to permit responses from any host.  Therefore implement it
// in terms of a ReadFrom, but discard the source address!
func (s mdnsShim) Read(b []byte) (n int, err error) {
	n, _, err = s.ReadFrom(b)
	return n, err
}

// RemoteAddr isn't very convincing for a listening socket, but it will do
func (s mdnsShim) RemoteAddr() net.Addr {
	return mdnsAddr
}

func InsertMdnsShim() {
	// Fudge resolver used by http client to do mdns lookups...
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := net.ListenUDP("udp", nil)
		if err != nil {
			return nil, err
		}
		return mdnsShim{conn}, nil
	}
}
