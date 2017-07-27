package service

import (
	//"barrage/matrix"
	"barrage/msg"
	"sync"
)

type GenericService struct {
	name  string
	serve func(msg.Message)
}

func (s *GenericService) Serve(msg msg.Message) {
	//scope := matrix.NewMatrixScope(s.name)
	s.serve(msg)
	//scope.Scope()
}

type ServiceFactory struct {
	services map[string]msg.Service
}

func CreateService(name string, fn func(msg.Message)) msg.Service {
	return &GenericService{
		name:  name,
		serve: fn,
	}
}

var (
	sf   *ServiceFactory
	once sync.Once
)

func Instance() *ServiceFactory {
	once.Do(func() {
		sf = &ServiceFactory{
			services: make(map[string]msg.Service),
		}
	})
	return sf
}

func (s *ServiceFactory) RegisterService(name string, fn func(msg.Message)) {
	s.services[name] = CreateService(name, fn)
}

func (s *ServiceFactory) GetService(name string) msg.Service {
	if serv, ok := s.services[name]; ok {
		return serv
	}
	return nil
}
