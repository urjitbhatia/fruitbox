package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestImages(t *testing.T) {
	proj := load(t, "basic")
	e := New(&runner.Fake{}, io.Discard)
	imgs := e.Images(proj)
	got := map[string]string{}
	for _, im := range imgs {
		got[im.Service] = im.Image
	}
	if got["web"] != "nginx:1.27" || got["db"] != "postgres:16" {
		t.Errorf("unexpected images: %v", got)
	}
}

func TestPortResolves(t *testing.T) {
	proj := load(t, "basic")
	e := New(&runner.Fake{}, io.Discard)
	mapped, err := e.Port(proj, "web", 80, "tcp")
	if err != nil {
		t.Fatalf("Port: %v", err)
	}
	if mapped != "0.0.0.0:8080" {
		t.Errorf("Port = %q, want 0.0.0.0:8080", mapped)
	}
	if _, err := e.Port(proj, "web", 9999, "tcp"); err == nil {
		t.Error("expected error for unmapped port")
	}
}

func TestCopyResolvesServiceRef(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	// Copy from the web container to a host path.
	if err := e.Copy(context.Background(), proj, "web:/etc/nginx/nginx.conf", "./out.conf", 1, false); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "cp basic-web-1:/etc/nginx/nginx.conf ./out.conf") == -1 {
		t.Errorf("service ref not resolved, calls: %v", calls)
	}
}

func TestParsePort(t *testing.T) {
	port, proto, err := ParsePort("53/udp")
	if err != nil || port != 53 || proto != "udp" {
		t.Errorf("ParsePort(53/udp) = %d %s %v", port, proto, err)
	}
	port, proto, _ = ParsePort("80")
	if port != 80 || proto != "tcp" {
		t.Errorf("ParsePort(80) = %d %s, want 80 tcp", port, proto)
	}
}

func TestCopyAllReplicas(t *testing.T) {
	proj := load(t, "basic")
	web, _ := proj.GetService("web")
	three := 3
	web.Scale = &three
	proj.Services["web"] = web

	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Copy(context.Background(), proj, "./conf", "web:/etc/conf", 1, true); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	calls := fake.CommandArgs()
	for n := 1; n <= 3; n++ {
		want := "cp ./conf basic-web-" + itoa(n) + ":/etc/conf"
		if firstMatch(calls, want) == -1 {
			t.Errorf("--all should copy to replica %d (%q), calls: %v", n, want, calls)
		}
	}
}
