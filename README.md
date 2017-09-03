# Superdiscoverer

A [Supervisor](http://supervisord.org/) backed service discoverer, which will automatically register and deregister services according to the corresponding event notifications sent by Supervisor.

Superdiscoverer supports pluggable service registries, which currently only includes [Consul](http://www.consul.io/).


## Installation

```bash
$ go get -u github.com/RussellLuo/superdiscoverer/cmd/superdiscoverer
```


## Quick start

### 1. Add `superdiscoverer` as an event listener

Suppose that:

- Your target service's process name in Supervisor is `test`, and it is listening on `127.0.0.1:8000`
- Use `consul` as an underlying service registry, and Consul is listening on `127.0.0.1:8500`

then a typical Supervisor configuration is as follows:

```
[eventlistener:superdiscoverer]
command=$GOPATH/bin/superdiscoverer --target=test@127.0.0.1:8000 --registrator=consul@127.0.0.1:8500
events=PROCESS_STATE_RUNNING,PROCESS_STATE_STOPPING
stdout_logfile=$LOGPATH/supervisor/logs/superdiscoverer/stdout.log
stderr_logfile=$LOGPATH/supervisor/logs/superdiscoverer/stderr.log
```

**NOTE**: Replace `$GOPATH` and `$LOGPATH` with the actual full path on your system.

See [here](http://supervisord.org/events.html) for more details about event notification protocol of Supervisor.

### 2. Update the event listener into your existing Supervisor

```bash
$ supervisorctl -c <supervisord.conf>
supervisor> update
superdiscoverer: added process group
```

### 3. Stop or start or restart the `test` process

```bash
supervisor> stop test
test: stopped
supervisor> start test
test: started
supervisor> restart test
test: stopped
test: started
```

### 4. Check out

- Check out the stderr log of `superdiscoverer` to see event notifications sent by Supervisor
- Check out the daemon log of Consul to see the behaviors of service registering/deregistering
- Check out the healthy services managed by Consul, by using its command-line interface or client APIs


## License

[MIT](http://opensource.org/licenses/MIT)
