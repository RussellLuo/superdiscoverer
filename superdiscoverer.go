package superdiscoverer

import (
	"log"
)

// Superdiscoverer is a Supervisor backed service discoverer,
// which will automatically registers and deregisters services
// according to the corresponding events notified by Supervisor.
type Superdiscoverer struct {
	eventListener  *EventListener
	targetServices []*Service
	registrator    Registrator
}

// New creates a superdiscoverer.
func New(eventlistener *EventListener, targetServices []*Service, registrator Registrator) *Superdiscoverer {
	return &Superdiscoverer{
		eventListener:  eventlistener,
		targetServices: targetServices,
		registrator:    registrator,
	}
}

// findTargetService finds the target service that matches the given event.
func (sd *Superdiscoverer) findTargetService(event *Event) *Service {
	for _, ts := range sd.targetServices {
		if ts.Name == event.GroupName+":"+event.ProcessName {
			return ts
		}
	}
	return nil
}

// Discover runs forever to listen for event notifications
// sent by Supervisor, until an error occurs.
func (sd *Superdiscoverer) Discover() error {
	for {
		headers, event, err := sd.eventListener.Wait()
		if err != nil {
			return err
		}
		log.Printf("headers: %+v, event: %+v\n", headers, event)

		go func() {
			service := sd.findTargetService(event)
			// Ignore the current event if the corresponding process
			// is not the target service
			if service == nil {
				return
			}

			switch event.Type {
			case EventTypeProcessStateRunning:
				// Register the current service *ONLY* when its state
				// has moved from STARTING to RUNNING
				if err := sd.registrator.Register(service); err != nil {
					log.Printf("Error: %v\n", err)
				}
			case EventTypeProcessStateStopping, EventTypeProcessStateExited:
				if event.FromState == FromStateRunning {
					// Deregister the current service *ONLY* when its state
					// has moved from RUNNING to STOPPING
					if err := sd.registrator.Deregister(service.ID()); err != nil {
						log.Printf("Error: %v\n", err)
					}
				}
			}
		}()

		if err := sd.eventListener.OK(); err != nil {
			return err
		}
	}
}
