[TOC]

# Summary
rulegen is a simple web server which can generate Prometheus rule file according
to parameters from the HTTP request. It a helper between [Linker DCOS client][client]
and [Prometheus][prom], it's used to monitor CPU/Mem usage of **host machines**.

# Clone
Clone to GOPATH
```sh
git clone https://bitbucket.org/linkernetworks/rulegen.git \
    $GOPATH/src/linkernetworks.com/dcos-backend/autoscaling/
```

# Build
## Build Local
Get dependencies
```sh
make get-dep
```

Build binary

```sh
make build
```

## Build Docker Image
```
make build-docker
```


# Run
## Run in Local
Set configurations in env var
```sh
# Set server listenning address (default: 127.0.0.1)
export LISTEN_ADDR=0.0.0.0
# Set server listenning port (default: 8080)
export LISTEN_PORT=8080
# Set rule file path (default: /linker/prometheus/generated/hostcpumem.rules)
export RULE_FILE=./dev.rules

make run
```

## Run in Docker Container
```sh
docker run -v /path/on/host:/linker/prometheus/generated \
 linkerrepository/rulegen:dev
```

# API
## Update Rules
**Request**
```http
PUT /rules HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Cache-Control: no-cache

{
	"cpu_enabled": true,
	"mem_enabled": true,
	"duration": "5m",
	"thresholds": {
		"cpu_high": 80,
		"cpu_low": 20,
		"mem_high": 90,
		"mem_low": 10
	}
}
```

**Response**
```json
{
    "success": true,
    "errmsg": ""
}
```

A rule file will be generated
```
ALERT HostHighCPUAlert
  IF host_cpu_usage > 80
  FOR 5m
  ANNOTATIONS {
    summary = "High CPU usage alert for host machine",
    description = "High CPU usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
  }

ALERT HostLowCPUAlert
  IF host_cpu_usage < 20
  FOR 5m
  ANNOTATIONS {
    summary = "Low CPU usage alert for host machine",
    description = "Low CPU usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
  }

ALERT HostHighMemAlert
  IF host_memory_usage > 90
  FOR 5m
  ANNOTATIONS {
    summary = "High memory usage alert for host machine",
    description = "High memory usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
  }

ALERT HostLowMemAlert
  IF host_memory_usage < 10
  FOR 5m
  ANNOTATIONS {
    summary = "Low memory usage alert for host machine",
    description = "Low memory usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
  }
```

If error occurs,
```json
{
    "success": false,
    "errmsg": "update rule: open /linker/prometheus/generated/hostcpumem.rules: no such file or directory"
}
```

# FAQ

[client]: https://bitbucket.org/linkernetworks/linker-dcos/src/577b8af558d974b2fb242fbfbda8a8a8bc72536f/client/?at=master
[prom]: https://bitbucket.org/linkernetworks/linker-dcos/src/577b8af558d974b2fb242fbfbda8a8a8bc72536f/docker/system/prometheus/?at=master
