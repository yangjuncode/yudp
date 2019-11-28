package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/yangjuncode/yudp"
)

import _ "net/http/pprof"

var port int
var policy int

func main() {
	flag.IntVar(&port, "port", 10000, "udp listen port")
	flag.IntVar(&policy, "policy", 0, "data slice new policy")

	flag.Parse()

	go func() {
		http.ListenAndServe(":8000", nil)
	}()

	opt := yudp.YudpOption{
		Addr:                "",
		Port:                port,
		ReusePort:           false,
		RecvBufSize:         1 * 1024 * 1024,
		SendBufSize:         1 * 1024 * 1024,
		UdpBatchSize:        64,
		RecvDataSlicePolicy: policy,
	}

	udpCon, err := yudp.NewListener(opt)
	if err != nil {
		log.Fatalln("udp new listener err:", err)
	}
	udpCon.Listen(func(data []byte, addr yudp.YudpAddr) {
		udpCon.WriteTo(data, &addr)

		if policy == 1 {
			yudp.PutDataSlice2Pool(data)
		}
	})
}
