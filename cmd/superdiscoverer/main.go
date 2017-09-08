package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	sd "github.com/RussellLuo/superdiscoverer"
	"github.com/RussellLuo/superdiscoverer/consul"
)

type StringArray []string

func (a *StringArray) Set(s string) error { *a = append(*a, s); return nil }
func (a *StringArray) String() string     { return strings.Join(*a, ",") }

func parseAddress(address string, errFmt string) (*sd.Service, error) {
	service := new(sd.Service)
	err := fmt.Errorf(errFmt, address)

	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return nil, err
	}
	service.Name = parts[0]

	hostport := strings.Split(parts[1], ":")
	if len(hostport) != 2 {
		return nil, err
	}
	service.Host = hostport[0]

	port, convErr := strconv.Atoi(hostport[1])
	if convErr != nil {
		return nil, err
	}
	service.Port = port

	return service, nil
}

func parseTargetAddress(address string) (*sd.Service, error) {
	errFmt := "Invalid --target `%v` (correct format: processname@host:port or groupname:processname@host:port)"
	service, err := parseAddress(address, errFmt)
	if err != nil {
		return nil, err
	}

	// processname@host:port => Name: processname:processname
	if !strings.ContainsAny(service.Name, ":") {
		service.Name = service.Name + ":" + service.Name
	}

	// Replace the local hostname ("localhost" or "127.0.0.1") with a meaningful hostname
	if service.Host == "localhost" || service.Host == "127.0.0.1" {
		hostname, sysErr := os.Hostname()
		if sysErr == nil {
			service.Host = hostname
		}
	}

	return service, nil
}

func main() {
	targetAddrs := StringArray{}
	flag.Var(&targetAddrs, "target", "target service address, e.g. processname@host:port (may be given multiple times)")
	registratorAddr := flag.String("registrator", "", "registrator address, e.g. consul@host:port")
	registratorConfig := flag.String("registrator-config", "", "registrator configuration in JSON format")
	flag.Parse()

	if len(targetAddrs) == 0 {
		log.Fatal("At least one --target is required")
	}
	targetServices := make([]*sd.Service, len(targetAddrs))
	for i, tAddr := range targetAddrs {
		s, err := parseTargetAddress(tAddr)
		if err != nil {
			log.Fatalf("Error: %v\n", err)
		}
		targetServices[i] = s
	}

	if *registratorAddr == "" {
		log.Fatal("--registrator is required")
	}
	r, err := parseAddress(*registratorAddr, "Invalid --registrator `%v` (correct format: consul@host:port)")
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	var registrator sd.Registrator
	switch r.Name {
	case "consul":
		msg := `Invalid --registrator-config, the full (and default) configurations of Consul-backed registrator are: '{"ttl":"3s","update_interval":"1s","deregister_interval":"1m"}', and you can specify a part of them.`
		config := new(consul.Config)
		if *registratorConfig != "" {
			if err := json.Unmarshal([]byte(*registratorConfig), config); err != nil {
				log.Fatal(msg)
			}
		}
		if err := config.Normalize(); err != nil {
			log.Fatal(msg)
		}
		registrator, err = consul.New(fmt.Sprintf("%v:%v", r.Host, r.Port), config)
	default:
		log.Fatalf("Unsupported registrator type `%v` specified in --registrator (now only support `consul`)", r.Name)
	}

	discoverer := sd.New(sd.StdEventListener, targetServices, registrator)
	if err := discoverer.Discover(); err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}
