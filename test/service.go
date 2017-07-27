package test

import "sync"

type Service struct {
	name string
	//serve func(Message)
	serve func()
}

func (s *Service) Serve() {
	s.serve()
}

type ServiceFactory struct {
	services map[string]*Service
}

func CreateService(name string, fn func() /*func(Message)*/) *Service {
	return &Service{
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
			services: make(map[string]*Service),
		}
	})
	return sf
}

func (s *ServiceFactory) RegisterService(name string, fn func() /*func(Message)*/) {
	s.services[name] = CreateService(name, fn)
}

func (s *ServiceFactory) GetService(name string) *Service {
	if serv, ok := s.services[name]; ok {
		return serv
	}
	return nil
}
