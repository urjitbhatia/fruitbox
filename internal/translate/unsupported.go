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

	// hostname, extra_hosts and sysctls are handled by fruitbox via generated
	// bind-mounts / post-start execs (see engine prepare step), so they are not
	// warned here. The following are genuine runtime/VM-isolation boundaries
	// that cannot currently be emulated.
	if svc.Privileged {
		add("privileged mode is not supported by the container runtime and will be ignored")
	}
	if svc.MacAddress != "" {
		add("mac_address is not supported by the container runtime and will be ignored")
	}
	if len(svc.Devices) > 0 {
		add("devices are not supported by the container runtime and will be ignored")
	}
	if len(svc.GroupAdd) > 0 {
		add("group_add is not supported by the container runtime and will be ignored")
	}

	return w
}
