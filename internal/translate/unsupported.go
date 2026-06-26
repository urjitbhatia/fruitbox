package translate

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// UnsupportedWarnings returns human-readable warnings for compose service
// attributes that Apple's `container run` cannot express, so behavior is
// transparent rather than silently dropped. The runtime, not fruitbox, is the
// limiting factor for these.
func UnsupportedWarnings(svc types.ServiceConfig) []string {
	var w []string
	add := func(format string, args ...any) {
		w = append(w, fmt.Sprintf(format, args...))
	}

	if svc.Hostname != "" {
		add("hostname %q is not supported by the container runtime and will be ignored", svc.Hostname)
	}
	if len(svc.ExtraHosts) > 0 {
		add("extra_hosts is not supported by the container runtime and will be ignored")
	}
	if svc.Privileged {
		add("privileged mode is not supported by the container runtime and will be ignored")
	}
	if svc.Restart != "" && svc.Restart != "no" {
		add("restart policy %q is not natively supported; fruitbox does not yet supervise restarts", svc.Restart)
	}
	if svc.MacAddress != "" {
		add("mac_address is not supported by the container runtime and will be ignored")
	}
	if len(svc.Sysctls) > 0 {
		add("sysctls are not supported by the container runtime and will be ignored")
	}
	if len(svc.Devices) > 0 {
		add("devices are not supported by the container runtime and will be ignored")
	}
	if len(svc.GroupAdd) > 0 {
		add("group_add is not supported by the container runtime and will be ignored")
	}

	return w
}
