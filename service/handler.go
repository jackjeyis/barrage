package service

import (
	"barrage/msg"
	//"barrage/logger"
	//"barrage/protocol"
)
type Handler interface {
	Handle(msg.Message)
	SetHandlerId(int)
	GetHandlerId() int
}

type ServiceHandler struct {
	handler_id int
}

func NewServiceHandler() *ServiceHandler {
	return &ServiceHandler{}
}

func (sh *ServiceHandler) SetHandlerId(id int) {
	sh.handler_id = id
}

func (sh *ServiceHandler) GetHandlerId() int {
	return sh.handler_id
}

func (sh *ServiceHandler) Handle(msg msg.Message) {
	msg.Channel().Serve(msg)
}

type IOHandler struct {
	handler_id int
}

func NewIOHandler() *IOHandler {
	return &IOHandler{}
}

func (ih *IOHandler) SetHandlerId(id int) {
	ih.handler_id = id
}

func (ih *IOHandler) GetHandlerId() int {
	return ih.handler_id
}

func (ih *IOHandler) Handle(msg msg.Message) {
	//logger.Info("handle msg %v",string(msg.(*protocol.Barrage).Body))
	msg.Channel().Send(msg)
}
