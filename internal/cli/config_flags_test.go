package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigNetworksImagesProfilesHash(t *testing.T) {
	file := filepath.Join("testdata", "basic", "compose.yaml")
	base := []string{"-f", file, "-p", "basic"}

	t.Run("networks", func(t *testing.T) {
		out, err := runRoot(t, append(base, "config", "--networks")...)
		if err != nil {
			t.Fatalf("err: %v\n%s", err, out)
		}
		// docker compose prints the network key, not the resolved name.
		if strings.TrimSpace(out) != "default" {
			t.Errorf("--networks = %q, want default", strings.TrimSpace(out))
		}
	})

	t.Run("images", func(t *testing.T) {
		out, err := runRoot(t, append(base, "config", "--images")...)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		got := strings.Fields(out)
		if len(got) != 2 || got[0] != "nginx:1.27" || got[1] != "postgres:16" {
			t.Errorf("--images = %v, want [nginx:1.27 postgres:16]", got)
		}
	})

	t.Run("hash all", func(t *testing.T) {
		out, err := runRoot(t, append(base, "config", "--hash", "*")...)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 2 {
			t.Fatalf("--hash * should print 2 lines, got %d: %q", len(lines), out)
		}
		for _, l := range lines {
			parts := strings.Fields(l)
			if len(parts) != 2 || len(parts[1]) != 64 {
				t.Errorf("bad hash line %q", l)
			}
		}
	})

	t.Run("no-interpolate still loads", func(t *testing.T) {
		if _, err := runRoot(t, append(base, "config", "--no-interpolate", "-q")...); err != nil {
			t.Errorf("--no-interpolate -q failed: %v", err)
		}
	})

	t.Run("output to file", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "out.yaml")
		if _, err := runRoot(t, append(base, "config", "-o", dst)...); err != nil {
			t.Fatalf("config -o: %v", err)
		}
		if _, err := readFile(dst); err != nil {
			t.Errorf("output file not written: %v", err)
		}
	})
}
