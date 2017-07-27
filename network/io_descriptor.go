package network

import (
	"barrage/logger"
	"barrage/protocol"
	"barrage/service"
	"fmt"
	"sync"
)

var (
	factory *IODescriptorFactory
	once    sync.Once
)

type IODescriptorFactory struct {
}

func Instance() *IODescriptorFactory {
	once.Do(func() {
		factory = &IODescriptorFactory{}
	})
	return factory
}

func (d *IODescriptorFactory) CreateAcceptor(io_service *service.IOService, addr, name string) (*Acceptor, error) {
	srv := service.Instance().GetService(name)
	address, proto_name := d.ParseAddress(addr)
	proto := protocol.Instance().Create(proto_name)
	a := NewAcceptor(io_service, srv, address, proto)
	return a, nil
}

func (d *IODescriptorFactory) ParseAddress(addr string) (address, proro_name string) {
	if _, err := fmt.Sscanf(addr, "%s%s", &address, &proro_name); err != nil {
		logger.Error("Sscanf error %v", err)
		return
	}
	return
}
