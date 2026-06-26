package translate

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestUnsupportedWarnings(t *testing.T) {
	svc := types.ServiceConfig{
		Name:       "web",
		Hostname:   "myhost",
		Privileged: true,
		Restart:    "always",
		ExtraHosts: types.HostsList{"a": []string{"1.2.3.4"}},
	}
	w := UnsupportedWarnings(svc)
	joined := strings.Join(w, "\n")
	for _, want := range []string{"hostname", "privileged", "restart", "extra_hosts"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected warning about %q, got:\n%s", want, joined)
		}
	}
}

func TestUnsupportedWarningsNoneForPlainService(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Image: "nginx", Restart: "no"}
	if w := UnsupportedWarnings(svc); len(w) != 0 {
		t.Errorf("expected no warnings, got %v", w)
	}
}
