// +build !linux

// udp_generic implements the UDP interface in pure Go stdlib. This
// means it can be used on platforms like Darwin and Windows.

package yudp

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type YudpAddr struct {
	net.UDPAddr
}

type YudpConn struct {
	*net.UDPConn
	Option YudpOption
}

type YudpHandler func(data []byte, addr YudpAddr)

func NewUDPAddr(ip uint32, port uint16) *YudpAddr {
	return &YudpAddr{
		UDPAddr: net.UDPAddr{
			IP:   int2ip(ip),
			Port: int(port),
		},
	}
}

func NewUDPAddrFromString(s string) *YudpAddr {
	p := strings.Split(s, ":")
	if len(p) < 2 {
		return nil
	}

	port, _ := strconv.Atoi(p[1])
	return &YudpAddr{
		UDPAddr: net.UDPAddr{
			IP:   net.ParseIP(p[0]),
			Port: port,
		},
	}
}

func NewListener(opt YudpOption) (*YudpConn, error) {
	lc := NewListenConfig(opt.ReusePort)
	pc, err := lc.ListenPacket(context.TODO(), "udp4", fmt.Sprintf("%s:%d", opt.Addr, opt.Port))
	if err != nil {
		return nil, err
	}
	if uc, ok := pc.(*net.UDPConn); ok {
		return &YudpConn{UDPConn: uc, Option: opt}, nil
	}
	return nil, fmt.Errorf("Unexpected PacketConn: %T %#v", pc, pc)
}

func (ua *YudpAddr) Equals(t *YudpAddr) bool {
	if t == nil || ua == nil {
		return t == nil && ua == nil
	}
	return ua.IP.Equal(t.IP) && ua.Port == t.Port
}

func (uc *YudpConn) WriteTo(b []byte, addr *YudpAddr) error {
	_, err := uc.UDPConn.WriteToUDP(b, &addr.UDPAddr)
	return err
}

func (uc *YudpConn) LocalAddr() (*YudpAddr, error) {
	a := uc.UDPConn.LocalAddr()

	switch v := a.(type) {
	case *net.UDPAddr:
		return &YudpAddr{UDPAddr: *v}, nil
	default:
		return nil, fmt.Errorf("LocalAddr returned: %#v", a)
	}
}

type rawMessage struct {
	Len uint32
}

func (u *YudpConn) Listen(handler YudpHandler) {
	buffer := make([]byte, YudpMTU)
	udpAddr := YudpAddr{}

	for {
		switch u.Option.RecvDataSlicePolicy {
		case 1:
			buffer = GetDataSliceFromPool()
		case 2:
		//no need processing
		case 0:
			fallthrough
		default:
			buffer = make([]byte, YudpMTU, YudpMTU)

		}
		// Just read one packet at a time
		n, rua, err := u.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		udpAddr.UDPAddr = *rua
		handler(buffer[:n], udpAddr)
	}
}

func udp2ip(addr *YudpAddr) net.IP {
	return addr.IP
}

func udp2ipInt(addr *YudpAddr) uint32 {
	return binary.BigEndian.Uint32(addr.IP.To4())
}

func hostDidRoam(addr *YudpAddr, newaddr *YudpAddr) bool {
	return !addr.Equals(newaddr)
}
