// Package utils contain implementations of everything that
// cannot be clearly classified in the context of this project.
package utils

import (
	"net"
	"strings"
)

// TCPAddrCmp compares 2 (ip, port) pair addresses and returns
// true if and only if they are equivalent.
func TCPAddrCmp(a, b string) bool {
	if ahost, aport, err := net.SplitHostPort(a); err == nil {
		if bhost, bport, err := net.SplitHostPort(b); err == nil {
			if net.ParseIP(ahost).Equal(net.ParseIP(bhost)) {
				return aport == bport
			}
		}
	}
	return false
}

// GetOutboundIP attempts to find the public IP of the
// outgoing TCP or UDP connections.
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx], nil
}
