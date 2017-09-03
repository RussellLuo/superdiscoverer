package superdiscoverer

import (
	"fmt"
)

type Service struct {
	Name string
	Host string
	Port int
}

func (s *Service) ID() string {
	return fmt.Sprintf("%v@%v:%v", s.Name, s.Host, s.Port)
}

type Registrator interface {
	Register(service *Service) error
	Deregister(serviceID string) error
}
