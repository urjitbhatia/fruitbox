package translate

import (
	"reflect"
	"testing"
)

func TestBuildImageTagDefaults(t *testing.T) {
	proj := loadProject(t, "build")
	api, _ := proj.GetService("api")
	if got := BuildImageTag(proj.Name, api); got != "build_api" {
		t.Errorf("api build tag = %q, want build_api", got)
	}
	worker, _ := proj.GetService("worker")
	if got := BuildImageTag(proj.Name, worker); got != "myorg/worker:latest" {
		t.Errorf("worker build tag = %q, want myorg/worker:latest (explicit image wins)", got)
	}
}

func TestBuildBuildArgs(t *testing.T) {
	proj := loadProject(t, "build")
	api, _ := proj.GetService("api")
	args := BuildBuildArgs(proj.Name, api, BuildExtra{})
	want := []string{
		"build",
		"--tag", "build_api",
		"--file", "Dockerfile.prod",
		"--target", "runtime",
		"--build-arg", "VERSION=1.2.3",
		// context is resolved to an absolute path by compose-go normalization,
		// so only assert it is the trailing argument below.
	}
	// Compare everything except the trailing context positional.
	if !reflect.DeepEqual(args[:len(want)], want) {
		t.Errorf("build args mismatch:\n got: %v\nwant prefix: %v", args, want)
	}
	if len(args) != len(want)+1 {
		t.Errorf("expected exactly one trailing context arg, got %v", args)
	}
}

func TestBuildRunArgsUsesBuiltTag(t *testing.T) {
	proj := loadProject(t, "build")
	api, _ := proj.GetService("api")
	args, err := BuildRunArgs(proj, api, RunOptions{Number: 1, Detach: true})
	if err != nil {
		t.Fatalf("BuildRunArgs: %v", err)
	}
	// The image positional must be the built tag, not empty.
	if args[len(args)-1] != "build_api" {
		t.Errorf("run image = %q, want build_api; args=%v", args[len(args)-1], args)
	}
}

func TestBuildBuildArgsNilWhenNoBuild(t *testing.T) {
	proj := loadProject(t, "basic")
	web, _ := proj.GetService("web")
	if args := BuildBuildArgs(proj.Name, web, BuildExtra{}); args != nil {
		t.Errorf("expected nil build args for image-only service, got %v", args)
	}
}
