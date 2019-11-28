package yudp

const YudpMTU = 1472

type YudpOption struct {
	//listen interface,default is empty,listen on all interface
	Addr string
	Port int
	//in linux this can be true, other os is not supported
	ReusePort bool
	//udp socket recv buf size,if not set, use os default
	RecvBufSize int
	//udp socket send buf size,if not set, use os default
	SendBufSize int
	//in linux this is used by recvmmsg(default:64), other os is not used
	UdpBatchSize int

	//0: new buffer every read,udpHandler no need additional processing for data parameter
	// in high pps programm, this will cause high gc
	//1: use yudp.pool, user should call PutDataSlice2Pool after processing
	//2: use a global data slice, so every udpHandler call will overwrite the data,useful for single thread processing
	RecvDataSlicePolicy int
}
