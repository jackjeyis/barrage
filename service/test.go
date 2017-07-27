package service

import (
	"fmt"
	"time"
)

type testfunc func(int)

type Protocol interface {
	Encode(msg Message) []byte
	Decode([]byte) Message
}
type Message interface {
	Protocol
}
type ServiceChannle struct {
	addr string
}

func (s *ServiceChannle) Run() {
	s.OnRead()
	s.OnWrite()
}
func (s *ServiceChannle) OnRead() {
	fmt.Println("OnRead ")
}

func (s *ServiceChannle) OnWrite() {
	fmt.Println("OnWrite ")
}

func test() {
	time.Sleep(2 * time.Second)
	fmt.Println("test for run")
}
func main() {
	serv := &ServiceChannle{}
	go func() {
		serv.addr = "sdfaad"
	}()

	time.Sleep(3 * time.Second)
	fmt.Println(serv.addr)
}
