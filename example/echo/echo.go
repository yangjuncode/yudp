package main

import (
	"log"

	"github.com/yangjuncode/yudp"
)

func main() {
	opt := yudp.YudpOption{
		Addr:         "",
		Port:         1000,
		ReusePort:    false,
		RecvBufSize:  1 * 1024 * 1024,
		SendBufSize:  1 * 1024 * 1024,
		UdpBatchSize: 64,
	}

	udpCon, err := yudp.NewListener(opt)
	log.Fatalln("udp new listener err:", err)
	udpCon.Listen(func(data []byte, addr yudp.YudpAddr) {
		udpCon.WriteTo(data, &addr)
	})
}
