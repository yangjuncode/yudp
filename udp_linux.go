package yudp

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type YudpConn struct {
	sysFd  int
	Option YudpOption
}

type YudpAddr struct {
	IP   uint32
	Port uint16
}

type YudpHandler func(data []byte, addr YudpAddr)

func NewUDPAddr(ip uint32, port uint16) *YudpAddr {
	return &YudpAddr{IP: ip, Port: port}
}

func NewUDPAddrFromString(s string) *YudpAddr {
	p := strings.Split(s, ":")
	if len(p) < 2 {
		return nil
	}

	port, _ := strconv.Atoi(p[1])
	return &YudpAddr{
		IP:   ip2int(net.ParseIP(p[0])),
		Port: uint16(port),
	}
}

type rawSockaddr struct {
	Family uint16
	Data   [14]uint8
}

type rawSockaddrAny struct {
	Addr rawSockaddr
	Pad  [96]int8
}

func NewListener(opt YudpOption) (*YudpConn, error) {
	syscall.ForkLock.RLock()
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err == nil {
		syscall.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()

	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if opt.UdpBatchSize == 0 {
		opt.UdpBatchSize = 64
	}

	udpCon := &YudpConn{
		sysFd:  fd,
		Option: opt,
	}

	if opt.RecvBufSize > 0 {
		udpCon.SetRecvBuffer(opt.RecvBufSize)
	}
	if opt.SendBufSize > 0 {
		udpCon.SetSendBuffer(opt.SendBufSize)
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, 0x0F, 1); err != nil {
		return nil, err
	}

	if err = syscall.Bind(fd, &syscall.SockaddrInet4{Port: opt.Port}); err != nil {
		return nil, err
	}

	// SO_REUSEADDR does not load balance so we use PORT
	if opt.ReusePort {
		if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
			return nil, err
		}
	}

	//TODO: this may be useful for forcing threads into specific cores
	//syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, unix.SO_INCOMING_CPU, x)
	//v, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, unix.SO_INCOMING_CPU)
	//l.Println(v, err)

	return udpCon, err
}

func (u *YudpConn) SetRecvBuffer(n int) error {
	return syscall.SetsockoptInt(u.sysFd, syscall.SOL_SOCKET, syscall.SO_RCVBUFFORCE, n)
}

func (u *YudpConn) SetSendBuffer(n int) error {
	return syscall.SetsockoptInt(u.sysFd, syscall.SOL_SOCKET, syscall.SO_SNDBUFFORCE, n)
}

func (u *YudpConn) GetRecvBuffer() (int, error) {
	return syscall.GetsockoptInt(int(u.sysFd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
}

func (u *YudpConn) GetSendBuffer() (int, error) {
	return syscall.GetsockoptInt(int(u.sysFd), syscall.SOL_SOCKET, syscall.SO_SNDBUF)
}

func (u *YudpConn) LocalAddr() (*YudpAddr, error) {
	var rsa rawSockaddrAny
	var rLen = syscall.SizeofSockaddrAny

	_, _, err := syscall.Syscall(
		syscall.SYS_GETSOCKNAME,
		uintptr(u.sysFd),
		uintptr(unsafe.Pointer(&rsa)),
		uintptr(unsafe.Pointer(&rLen)),
	)

	if err != 0 {
		return nil, err
	}

	addr := &YudpAddr{}
	if rsa.Addr.Family == syscall.AF_INET {
		addr.Port = uint16(rsa.Addr.Data[0])<<8 + uint16(rsa.Addr.Data[1])
		addr.IP = uint32(rsa.Addr.Data[2])<<24 + uint32(rsa.Addr.Data[3])<<16 + uint32(rsa.Addr.Data[4])<<8 + uint32(rsa.Addr.Data[5])
	} else {
		addr.Port = 0
		addr.IP = 0
	}
	return addr, nil
}

func (u *YudpConn) Listen(handler YudpHandler) {
	udpAddr := YudpAddr{}

	//TODO: should we track this?
	//metric := metrics.GetOrRegisterHistogram("test.batch_read", nil, metrics.NewExpDecaySample(1028, 0.015))
	msgs, buffers, names := u.PrepareRawMessages(u.Option.UdpBatchSize)

	for {
		msgs, buffers, names = adjustMsgs(msgs, buffers, names, u.Option.RecvDataSlicePolicy)

		n, err := u.ReadMulti(msgs)
		if err != nil {
			continue
		}

		//metric.Update(int64(n))
		for i := 0; i < n; i++ {
			udpAddr.IP = binary.BigEndian.Uint32(names[i][4:8])
			udpAddr.Port = binary.BigEndian.Uint16(names[i][2:4])

			handler(buffers[i][:msgs[i].Len], udpAddr)

		}
	}
}

func (u *YudpConn) ReadMulti(msgs []rawMessage) (int, error) {
	for {
		n, _, err := syscall.Syscall6(
			syscall.SYS_RECVMMSG,
			uintptr(u.sysFd),
			uintptr(unsafe.Pointer(&msgs[0])),
			uintptr(len(msgs)),
			unix.MSG_WAITFORONE,
			0,
			0,
		)

		if err != 0 {
			return 0, &net.OpError{Op: "recvmmsg", Err: err}
		}

		return int(n), nil
	}
}

func (u *YudpConn) WriteTo(b []byte, addr *YudpAddr) error {
	var rsa syscall.RawSockaddrInet4

	//TODO: sometimes addr is nil!
	rsa.Family = syscall.AF_INET
	p := (*[2]byte)(unsafe.Pointer(&rsa.Port))
	p[0] = byte(addr.Port >> 8)
	p[1] = byte(addr.Port)

	rsa.Addr[0] = byte(addr.IP & 0xff000000 >> 24)
	rsa.Addr[1] = byte(addr.IP & 0x00ff0000 >> 16)
	rsa.Addr[2] = byte(addr.IP & 0x0000ff00 >> 8)
	rsa.Addr[3] = byte(addr.IP & 0x000000ff)

	for {
		_, _, err := syscall.Syscall6(
			syscall.SYS_SENDTO,
			uintptr(u.sysFd),
			uintptr(unsafe.Pointer(&b[0])),
			uintptr(len(b)),
			uintptr(0),
			uintptr(unsafe.Pointer(&rsa)),
			uintptr(syscall.SizeofSockaddrInet4),
		)

		if err != 0 {
			return &net.OpError{Op: "sendto", Err: err}
		}

		//TODO: handle incomplete writes

		return nil
	}
}

func (ua *YudpAddr) Equals(t *YudpAddr) bool {
	if t == nil || ua == nil {
		return t == nil && ua == nil
	}
	return ua.IP == t.IP && ua.Port == t.Port
}

func (ua *YudpAddr) Copy() *YudpAddr {
	return &YudpAddr{
		Port: ua.Port,
		IP:   ua.IP,
	}
}

func (ua *YudpAddr) String() string {
	return fmt.Sprintf("%s:%v", int2ip(ua.IP), ua.Port)
}

func udp2ip(addr *YudpAddr) net.IP {
	return int2ip(addr.IP)
}

func udp2ipInt(addr *YudpAddr) uint32 {
	return addr.IP
}

func hostDidRoam(addr *YudpAddr, newaddr *YudpAddr) bool {
	return !addr.Equals(newaddr)
}
