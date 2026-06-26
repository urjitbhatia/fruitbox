package translate

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestUnsupportedWarnings(t *testing.T) {
	svc := types.ServiceConfig{
		Name:       "web",
		Privileged: true,
		MacAddress: "02:42:ac:11:00:02",
		GroupAdd:   []string{"audio"},
	}
	w := UnsupportedWarnings(svc)
	joined := strings.Join(w, "\n")
	for _, want := range []string{"privileged", "mac_address", "group_add"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected warning about %q, got:\n%s", want, joined)
		}
	}
}

func TestUnsupportedWarningsNoneForHandledAttrs(t *testing.T) {
	// hostname, extra_hosts and sysctls are handled via workarounds, not warned.
	svc := types.ServiceConfig{
		Name:       "web",
		Image:      "nginx",
		Restart:    "always",
		Hostname:   "myhost",
		ExtraHosts: types.HostsList{"a": []string{"1.2.3.4"}},
		Sysctls:    types.Mapping{"net.core.somaxconn": "1024"},
	}
	if w := UnsupportedWarnings(svc); len(w) != 0 {
		t.Errorf("expected no warnings for handled attrs, got %v", w)
	}
}
