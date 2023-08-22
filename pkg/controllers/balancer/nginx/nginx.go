package nginx

import (
	"fmt"
	balancerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"
	"strings"
)

type server struct {
	name     string
	protocol string
	port     int32
	upstream string // server.upstream will be processed exactly by an upstream
}

func (s *server) conf() string {
	var protocol string
	if s.protocol == "udp" {
		protocol = "udp"
	}
	return fmt.Sprintf(`
server {
    listen %d %s;
    proxy_pass %s;
}
`, s.port, protocol, s.upstream)
}

// backend is a backend service.
type backend struct {
	name   string
	weight int32
}

// upstream acts as the value of the key `proxy_pass` in nginx.conf.
type upstream struct {
	name     string
	backends []backend
	port     int32
}

// conf returns the config segment for the key `upstream` in nginx.conf.
// Example：
//
//	upstream upstream_http {
//	    server example-controller-v1-backend:80 weight=40;
//	    server example-controller-v2-backend:80 weight=20;
//	    server example-controller-v3-backend:80 weight=40;
//	}
func (us *upstream) conf() string {
	backendStr := ""
	for _, b := range us.backends {
		backendStr += fmt.Sprintf("    server %s:%d weight=%d;\n", b.name, us.port, b.weight)
	}
	return fmt.Sprintf(`
upstream %s {
%s
}
`, us.name, backendStr)
}

// NewConfig generates the `nginx.conf` with the given Balancer instance.
// Example:
// ===================== nginx.conf =====================
//
//	events {
//	    worker_connections 1024;
//	}
//
//	stream {
//	    server {
//	        listen 80 tcp;
//	        proxy_pass upstream_http;
//	    }
//	    upstream upstream_http {
//	        server example-controller-v1-backend:80 weight=20;
//	        server example-controller-v2-backend:80 weight=80;
//	    }
//	}
//
// ======================================================
func NewConfig(balancer *balancerv1.Balancer) string {
	var servers []server
	for _, balancerPort := range balancer.Spec.Ports {
		servers = append(servers, server{
			name:     balancerPort.Name,
			protocol: strings.ToLower(string(balancerPort.Protocol)),
			port:     int32(balancerPort.Port),
			upstream: fmt.Sprintf("upstream_%s", balancerPort.Name),
		})
	}

	var backends []backend
	for _, balancerBackend := range balancer.Spec.Backends {
		backends = append(backends, backend{
			name:   fmt.Sprintf("%s-%s-backend", balancer.Name, balancerBackend.Name),
			weight: balancerBackend.Weight,
		})
	}

	var upstreams []upstream
	for _, s := range servers {
		upstreams = append(upstreams, upstream{
			name:     s.upstream,
			backends: backends,
			port:     s.port,
		})
	}

	conf := ""
	conf += "events {\n"
	conf += "    worker_connections 1024;\n"
	conf += "}\n"
	conf += "stream {\n"

	for _, s := range servers {
		conf += s.conf()
	}

	for _, us := range upstreams {
		conf += us.conf()
	}

	conf += "}\n"

	return conf
}
