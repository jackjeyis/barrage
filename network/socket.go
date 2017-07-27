package network

import (
	"barrage/protocol"
	"net"
)

type Socket struct {
	conn     *net.TCPConn
	protocol protocol.Protocol
}
