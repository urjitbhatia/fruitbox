package translate

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestHostnameFileContent(t *testing.T) {
	if got := HostnameFileContent(types.ServiceConfig{Hostname: "web1"}); got != "web1\n" {
		t.Errorf("HostnameFileContent = %q, want web1\\n", got)
	}
	if got := HostnameFileContent(types.ServiceConfig{}); got != "" {
		t.Errorf("expected empty hostname content, got %q", got)
	}
}

func TestHostsFileContent(t *testing.T) {
	svc := types.ServiceConfig{ExtraHosts: types.HostsList{
		"db.local":  []string{"10.0.0.5"},
		"api.local": []string{"10.0.0.6", "10.0.0.7"},
	}}
	got := HostsFileContent(svc)
	if !strings.Contains(got, "127.0.0.1\tlocalhost") {
		t.Errorf("missing loopback entry:\n%s", got)
	}
	for _, want := range []string{"10.0.0.5\tdb.local", "10.0.0.6\tapi.local", "10.0.0.7\tapi.local"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing host entry %q in:\n%s", want, got)
		}
	}
	if HostsFileContent(types.ServiceConfig{}) != "" {
		t.Error("expected empty hosts content when no extra_hosts")
	}
}

func TestSysctlExecArgs(t *testing.T) {
	svc := types.ServiceConfig{Sysctls: types.Mapping{
		"net.core.somaxconn":      "1024",
		"net.ipv4.tcp_syncookies": "1",
	}}
	got := SysctlExecArgs("web-1", svc)
	if len(got) != 2 {
		t.Fatalf("expected 2 sysctl execs, got %d: %v", len(got), got)
	}
	// Sorted by key: net.core.somaxconn first.
	if strings.Join(got[0], " ") != "exec web-1 sysctl -w net.core.somaxconn=1024" {
		t.Errorf("unexpected first sysctl exec: %v", got[0])
	}
}
