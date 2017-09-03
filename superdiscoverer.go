package superdiscoverer

import (
	"log"
)

// Superdiscoverer is a Supervisor backed service discoverer,
// which will automatically registers and deregisters services
// according to the corresponding events notified by Supervisor.
type Superdiscoverer struct {
	eventListener  *EventListener
	targetServices []Service
	registrator    Registrator
}

// New creates a superdiscoverer.
func New(eventlistener *EventListener, targetServices []Service, registrator Registrator) *Superdiscoverer {
	return &Superdiscoverer{
		eventListener:  eventlistener,
		targetServices: targetServices,
		registrator:    registrator,
	}
}

// Discover runs the superdiscoverer forever to listen to the event
// notifications from Supervisor, until an error occurs.
func (sd *Superdiscoverer) Discover() error {
	for {
		headers, event, err := sd.eventListener.Wait()
		if err != nil {
			return err
		}
		log.Printf("headers: %+v, event: %+v\n", headers, event)

		go func() {
			var service *Service
			for _, ts := range sd.targetServices {
				if ts.Name == event.ProcessName {
					service = &ts
					break
				}
			}
			// Ignore the current process if it is not the target service
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
			case EventTypeProcessStateStopping:
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
