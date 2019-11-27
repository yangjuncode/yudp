package yudp

// Windows support is primarily implemented in udp_generic, besides NewListenConfig

import (
	"fmt"
	"net"
	"syscall"
)

func NewListenConfig(reusePort bool) net.ListenConfig {
	return net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			if reusePort {
				// There is no way to support multiple listeners safely on Windows:
				// https://docs.microsoft.com/en-us/windows/desktop/winsock/using-so-reuseaddr-and-so-exclusiveaddruse
				return fmt.Errorf("reusePort not supported on windows")
			}
			return nil
		},
	}
}
