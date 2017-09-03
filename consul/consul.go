package consul

import (
	"log"
	"sync"
	"time"

	sd "github.com/RussellLuo/superdiscoverer"
	"github.com/hashicorp/consul/api"
)

// Consul is a Consul backed service registrator implementation.
type Consul struct {
	agent  *api.Agent
	config *Config

	mu          sync.Mutex
	deregisters map[string]func() error
}

type Config struct {
	TTL                            string `json:"ttl,omitempty"`
	UpdateTTLInterval              string `json:"update_interval,omitempty"`
	DeregisterCriticalServiceAfter string `json:"deregister_interval,omitempty"`

	dTTL                            time.Duration `json:"-"`
	dUpdateTTLInterval              time.Duration `json:"-"`
	dDeregisterCriticalServiceAfter time.Duration `json:"-"`
}

func (c *Config) Normalize() error {
	if c.TTL == "" {
		c.TTL = "3s"
	}
	if c.UpdateTTLInterval == "" {
		c.UpdateTTLInterval = "1s"
	}
	if c.DeregisterCriticalServiceAfter == "" {
		c.DeregisterCriticalServiceAfter = "1m"
	}

	var err error
	if c.dTTL, err = time.ParseDuration(c.TTL); err != nil {
		return err
	}
	if c.dUpdateTTLInterval, err = time.ParseDuration(c.UpdateTTLInterval); err != nil {
		return err
	}
	if c.dDeregisterCriticalServiceAfter, err = time.ParseDuration(c.DeregisterCriticalServiceAfter); err != nil {
		return err
	}

	return nil
}

func New(address string, config *Config) (*Consul, error) {
	client, err := api.NewClient(&api.Config{Address: address})
	if err != nil {
		return nil, err
	}

	log.Printf("Creating a Consul (@%v) backed service registrator with configuration %+v\n", address, config)
	return &Consul{
		agent:       client.Agent(),
		config:      config,
		deregisters: make(map[string]func() error),
	}, nil
}

// Register registers the current service to Consul.
func (c *Consul) Register(service *sd.Service) error {
	var ()

	s := &api.AgentServiceRegistration{
		ID:      service.ID(),
		Name:    service.Name,
		Tags:    []string{},
		Address: service.Host,
		Port:    service.Port,
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: c.config.dDeregisterCriticalServiceAfter.String(),
			TTL: c.config.dTTL.String(),
		},
	}
	err := c.agent.ServiceRegister(s)
	if err != nil {
		return err
	}

	closing := make(chan struct{})
	closed := make(chan struct{})

	// Updates TTL asynchronously
	go func() {
		t := time.NewTicker(c.config.dUpdateTTLInterval)
		defer t.Stop()

		for {
			select {
			case <-closing:
				close(closed)
				return
			case <-t.C:
				if err := c.agent.UpdateTTL("service:"+service.ID(), "", api.HealthPassing); err != nil {
					log.Printf("Error: %v\n", err)
				}
			}
		}
	}()

	c.mu.Lock()
	c.deregisters[service.ID()] = func() error {
		if err := c.agent.ServiceDeregister(service.ID()); err != nil {
			return err
		}
		close(closing)
		<-closed
		return nil
	}
	c.mu.Unlock()

	return nil
}

// Deregister deregisters the current service from Consul.
func (c *Consul) Deregister(serviceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	deregister, ok := c.deregisters[serviceID]
	if !ok {
		return nil
	}

	if err := deregister(); err != nil {
		return err
	}
	// Remove the current deregister from the map only after deregistering succeeds.
	delete(c.deregisters, serviceID)
	return nil
}
