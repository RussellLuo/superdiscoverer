package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"encoding/json"
	sd "github.com/RussellLuo/superdiscoverer"
	"github.com/RussellLuo/superdiscoverer/consul"
)

type StringArray []string

func (a *StringArray) Set(s string) error { *a = append(*a, s); return nil }
func (a *StringArray) String() string     { return strings.Join(*a, ",") }

func parseAddress(address string, replaceLocalHostname bool) (sd.Service, error) {
	var service sd.Service
	err := fmt.Errorf("Invalid address `%v` (correct format: name@host:port)", address)

	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return service, err
	}
	service.Name = parts[0]

	hostport := strings.Split(parts[1], ":")
	if len(hostport) != 2 {
		return service, err
	}
	service.Host = hostport[0]

	if replaceLocalHostname {
		// Replace the local hostname ("localhost" or "127.0.0.1") with a meaningful hostname
		if service.Host == "localhost" || service.Host == "127.0.0.1" {
			hostname, sysErr := os.Hostname()
			if sysErr == nil {
				service.Host = hostname
			}
		}
	}

	port, convErr := strconv.Atoi(hostport[1])
	if convErr != nil {
		return service, err
	}
	service.Port = port

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
	targetServices := make([]sd.Service, len(targetAddrs))
	for i, tAddr := range targetAddrs {
		s, err := parseAddress(tAddr, true)
		if err != nil {
			log.Fatalf("Error: %v\n", err)
		}
		targetServices[i] = s
	}

	if *registratorAddr == "" {
		log.Fatal("--registrator is required")
	}
	r, err := parseAddress(*registratorAddr, false)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	var registrator sd.Registrator
	switch r.Name {
	case "consul":
		info := `Invalid --registrator-config, the full (and default) configurations of Consul-backed registrator are: '{"ttl":"3s","update_interval":"1s","deregister_interval":"1m"}', and you can specify a part of them.`
		config := new(consul.Config)
		if *registratorConfig != "" {
			if err := json.Unmarshal([]byte(*registratorConfig), config); err != nil {
				log.Fatal(info)
			}
		}
		if err := config.Normalize(); err != nil {
			log.Fatal(info)
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
