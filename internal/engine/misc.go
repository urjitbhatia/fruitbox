package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// ImageInfo describes the image a service uses.
type ImageInfo struct {
	Service string
	Image   string
}

// Images returns the image used by each service, sorted by service name.
func (e *Engine) Images(p *types.Project) []ImageInfo {
	var out []ImageInfo
	for _, name := range sortedServiceNames(p) {
		svc, err := p.GetService(name)
		if err != nil {
			continue
		}
		img := svc.Image
		if img == "" && svc.Build != nil {
			img = translate.BuildImageTag(p.Name, svc)
		}
		out = append(out, ImageInfo{Service: name, Image: img})
	}
	return out
}

// Port resolves the published host port for a service's container port.
// It returns an error if no matching published port exists.
func (e *Engine) Port(p *types.Project, service string, private int, protocol string) (string, error) {
	svc, err := p.GetService(service)
	if err != nil {
		return "", err
	}
	if protocol == "" {
		protocol = "tcp"
	}
	for _, port := range svc.Ports {
		if int(port.Target) != private {
			continue
		}
		proto := port.Protocol
		if proto == "" {
			proto = "tcp"
		}
		if proto != protocol {
			continue
		}
		if port.Published == "" {
			return "", fmt.Errorf("port %d/%s is exposed but not published", private, protocol)
		}
		host := port.HostIP
		if host == "" {
			host = "0.0.0.0"
		}
		return host + ":" + port.Published, nil
	}
	return "", fmt.Errorf("no published port for %d/%s in service %q", private, protocol, service)
}

// Copy copies files between the host and a service container. Either src or
// dest may be of the form "SERVICE:PATH"; that side is resolved to the
// service's container name before delegating to `container cp`.
func (e *Engine) Copy(ctx context.Context, p *types.Project, src, dest string, index int) error {
	if index <= 0 {
		index = 1
	}
	rsrc, err := e.resolveCopyPath(p, src, index)
	if err != nil {
		return err
	}
	rdest, err := e.resolveCopyPath(p, dest, index)
	if err != nil {
		return err
	}
	_, err = e.Runner.Run(ctx, "cp", rsrc, rdest)
	return err
}

// resolveCopyPath rewrites a SERVICE:PATH reference to CONTAINER:PATH. Plain
// host paths (including absolute paths with no service prefix) pass through.
func (e *Engine) resolveCopyPath(p *types.Project, path string, index int) (string, error) {
	prefix, rest, ok := strings.Cut(path, ":")
	if !ok {
		return path, nil
	}
	svc, err := p.GetService(prefix)
	if err != nil {
		// Not a service reference (e.g. a Windows drive path); pass through.
		return path, nil
	}
	return containerName(p, svc, index) + ":" + rest, nil
}

// ParsePort splits a "PORT[/PROTO]" argument into its numeric port and protocol.
func ParsePort(arg string) (int, string, error) {
	portStr, proto, _ := strings.Cut(arg, "/")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid port %q", arg)
	}
	if proto == "" {
		proto = "tcp"
	}
	return port, proto, nil
}

func sortedServiceNames(p *types.Project) []string {
	names := p.ServiceNames()
	sort.Strings(names)
	return names
}
