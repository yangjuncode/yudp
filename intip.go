package yudp

import (
	"encoding/binary"
	"fmt"
	"net"
)

// A helper type to avoid converting to IP when logging
type IntIp uint32

func (ip IntIp) String() string {
	return fmt.Sprintf("%v", int2ip(uint32(ip)))
}

func (ip IntIp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", int2ip(uint32(ip)).String())), nil
}

func ip2int(ip []byte) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
