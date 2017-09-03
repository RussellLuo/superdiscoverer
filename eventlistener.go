package superdiscoverer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	READY_FOR_EVENTS_TOKEN = "READY\n"
	RESULT_TOKEN_START     = "RESULT "

	EventTypeProcessStateRunning  = "PROCESS_STATE_RUNNING"
	EventTypeProcessStateStopping = "PROCESS_STATE_STOPPING"

	FromStateRunning = "RUNNING"
)

func atoi(s string, i *int) error {
	tempI, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i = tempI
	return nil
}

// header represents a Supervisor event notification header.
type header struct {
	Ver        string
	Server     string
	Serial     int
	Pool       string
	PoolSerial int
	EventName  string
	Len        int
}

// newEvent creates an header from given header tokens.
func newHeader(tokens map[string]string) (*header, error) {
	h := new(header)
	for key, value := range tokens {
		switch key {
		case "ver":
			h.Ver = value
		case "server":
			h.Server = value
		case "serial":
			if err := atoi(value, &h.Serial); err != nil {
				return nil, err
			}
		case "pool":
			h.Pool = value
		case "poolserial":
			if err := atoi(value, &h.PoolSerial); err != nil {
				return nil, err
			}
		case "eventname":
			h.EventName = value
		case "len":
			if err := atoi(value, &h.Len); err != nil {
				return nil, err
			}
		}
	}
	return h, nil
}

// Event represents a Supervisor event payload, whose event type
// is PROCESS_STATE_RUNNING or PROCESS_STATE_STOPPING, we only
// need to care about these two event types for automatic service-discovery.
type Event struct {
	Type        string
	ProcessName string
	GroupName   string
	FromState   string
	Pid         int
}

// newEvent creates an event from given eventName and payload tokens.
func newEvent(eventName string, tokens map[string]string) (*Event, error) {
	event := &Event{Type: eventName}
	for key, value := range tokens {
		switch key {
		case "processname":
			event.ProcessName = value
		case "groupname":
			event.GroupName = value
		case "from_state":
			event.FromState = value
		case "pid":
			if err := atoi(value, &event.Pid); err != nil {
				return nil, err
			}
		}
	}
	return event, nil
}

// EventListener is an event listener implementation of Supervisor.
// See http://supervisord.org/events.html for the protocol specification,
// and see https://github.com/Supervisor/supervisor/blob/master/supervisor/childutils.py
// for the similar implementation (class EventListenerProtocol) in Python.
type EventListener struct {
	Reader *bufio.Reader
	Writer *bufio.Writer
}

// StdEventListener is a standard event listener,
// which is bound to os.Stdin and os.Stdout.
var StdEventListener = &EventListener{
	Reader: bufio.NewReader(os.Stdin),
	Writer: bufio.NewWriter(os.Stdout),
}

// getTokens parses tokens (i.e. key-value pairs) from the given content.
func (el *EventListener) getTokens(content string) map[string]string {
	tokens := strings.Split(content, " ")
	kvPairs := make(map[string]string, len(tokens))
	for _, token := range tokens {
		pair := strings.Split(token, ":")
		kvPairs[pair[0]] = pair[1]
	}
	return kvPairs
}

// getHeaders parses the "header" line on the listener's stdin into a header.
func (el *EventListener) getHeader() (*header, error) {
	line, _, err := el.Reader.ReadLine()
	if err != nil {
		return nil, err
	}
	h, err := newHeader(el.getTokens(string(line)))
	if err != nil {
		return nil, err
	}
	return h, nil
}

// getPayload parses the payload on the listener's stdin into an event.
func (el *EventListener) getEvent(eventName string, length int) (*Event, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(el.Reader, bytes); err != nil {
		return nil, err
	}
	event, err := newEvent(eventName, el.getTokens(string(bytes)))
	if err != nil {
		return nil, err
	}
	return event, nil
}

// send writes and flushes bytes to the listener's stdout.
func (el *EventListener) send(bytes []byte) error {
	if _, err := el.Writer.Write(bytes); err != nil {
		return err
	}
	if err := el.Writer.Flush(); err != nil {
		return err
	}
	return nil
}

// ready sends a READY token to the listener's stdout.
func (el *EventListener) ready() error {
	return el.send([]byte(READY_FOR_EVENTS_TOKEN))
}

// result sends a result token to the listener's stdout.
func (el *EventListener) result(data string) error {
	resultToken := fmt.Sprintf("%s%d\n%s", RESULT_TOKEN_START, len(data), data)
	return el.send([]byte(resultToken))
}

// Wait waits for an event notification from Supervisor.
func (el *EventListener) Wait() (*header, *Event, error) {
	if err := el.ready(); err != nil {
		return nil, nil, err
	}

	header, err := el.getHeader()
	if err != nil {
		return nil, nil, err
	}

	event, err := el.getEvent(header.EventName, header.Len)
	if err != nil {
		return nil, nil, err
	}

	return header, event, nil
}

// OK sends a OK result token to the listener's stdout.
func (el *EventListener) OK() error {
	return el.result("OK")
}

// Fail sends a FAIL result token to the listener's stdout.
func (el *EventListener) Fail() error {
	return el.result("FAIL")
}
