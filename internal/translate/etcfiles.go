package translate

import (
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// HostnameFileContent returns the desired /etc/hostname contents for a service,
// or "" if no hostname is set. Apple's container runtime has no --hostname flag,
// so fruitbox bind-mounts this file instead.
func HostnameFileContent(svc types.ServiceConfig) string {
	if svc.Hostname == "" {
		return ""
	}
	return svc.Hostname + "\n"
}

// HostsFileContent returns the desired /etc/hosts contents for a service when
// extra_hosts is set, or "" otherwise. Standard loopback entries are included
// so the file is a complete, mountable /etc/hosts. The runtime exposes no
// --add-host flag, so fruitbox bind-mounts this file.
func HostsFileContent(svc types.ServiceConfig) string {
	if len(svc.ExtraHosts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("127.0.0.1\tlocalhost\n")
	b.WriteString("::1\tlocalhost ip6-localhost ip6-loopback\n")

	// ExtraHosts maps hostname -> []IP. Emit one line per IP, sorted by host
	// then IP for deterministic output.
	hosts := make([]string, 0, len(svc.ExtraHosts))
	for h := range svc.ExtraHosts {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	for _, h := range hosts {
		ips := append([]string(nil), svc.ExtraHosts[h]...)
		sort.Strings(ips)
		for _, ip := range ips {
			b.WriteString(ip)
			b.WriteString("\t")
			b.WriteString(h)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// SysctlExecArgs returns the `container exec` argument vectors needed to apply
// each namespaced sysctl after the container starts (the runtime has no
// --sysctl flag). Each returned slice is a full exec command sans binary.
func SysctlExecArgs(containerName string, svc types.ServiceConfig) [][]string {
	if len(svc.Sysctls) == 0 {
		return nil
	}
	keys := make([]string, 0, len(svc.Sysctls))
	for k := range svc.Sysctls {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out [][]string
	for _, k := range keys {
		out = append(out, []string{"exec", containerName, "sysctl", "-w", k + "=" + svc.Sysctls[k]})
	}
	return out
}
