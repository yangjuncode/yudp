package yudp

import "sync"

var dataSlicePool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, mtu)
	},
}

func PutDataSlice2Pool(dataBuf []byte) {
	dataSlicePool.Put(dataBuf)
}

func GetDataSliceFromPool() []byte {
	return dataSlicePool.Get().([]byte)
}
