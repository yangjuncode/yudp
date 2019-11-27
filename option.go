package yudp

const mtu = 9001

type YudpOption struct {
	Addr         string
	Port         int
	ReusePort    bool
	RecvBufSize  int
	SendBufSize  int
	UdpBatchSize int
}
