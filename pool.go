package yudp

import "sync"

var dataSlicePool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, mtu,mtu)
	},
}

func PutDataSlice2Pool(dataBuf []byte) {
	//dataBuf = dataBuf[:cap(dataBuf)]
	dataSlicePool.Put(dataBuf)
}

func GetDataSliceFromPool() []byte {
	return dataSlicePool.Get().([]byte)
}
