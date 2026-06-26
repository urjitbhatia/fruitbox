package translate

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/compose"
)

func loadProject(t *testing.T, dir string) *composeProject {
	t.Helper()
	proj, err := compose.Load(context.Background(), compose.LoadOptions{
		ConfigPaths: []string{filepath.Join("testdata", dir, "compose.yaml")},
		ProjectName: dir,
	})
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	return proj
}

func TestContainerName(t *testing.T) {
	if got := ContainerName("basic", "web", 1); got != "basic-web-1" {
		t.Errorf("ContainerName = %q, want basic-web-1", got)
	}
	if got := ContainerName("my proj", "db", 2); got != "myproj-db-2" {
		t.Errorf("ContainerName = %q, want myproj-db-2 (sanitized)", got)
	}
}

func TestBuildRunArgsWeb(t *testing.T) {
	proj := loadProject(t, "basic")
	svc, err := proj.GetService("web")
	if err != nil {
		t.Fatal(err)
	}
	args, err := BuildRunArgs(proj, svc, RunOptions{Number: 1, Detach: true})
	if err != nil {
		t.Fatalf("BuildRunArgs: %v", err)
	}
	want := []string{
		"run",
		"--name", "basic-web-1",
		"--detach",
		"--network", "basic_net",
		"--env", "GREETING=hello",
		"--publish", "8080:80",
		"--label", "com.docker.compose.container-number=1",
		"--label", "com.docker.compose.oneoff=False",
		"--label", "com.docker.compose.project=basic",
		"--label", "com.docker.compose.service=web",
		"nginx:1.27",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("BuildRunArgs(web) mismatch:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildRunArgsDbVolume(t *testing.T) {
	proj := loadProject(t, "basic")
	svc, err := proj.GetService("db")
	if err != nil {
		t.Fatal(err)
	}
	args, err := BuildRunArgs(proj, svc, RunOptions{Number: 1, Detach: true})
	if err != nil {
		t.Fatalf("BuildRunArgs: %v", err)
	}
	// The named volume "dbdata" must be resolved to its project-scoped name.
	if !containsPair(args, "--volume", "basic_dbdata:/var/lib/postgresql/data") {
		t.Errorf("db run args missing resolved named volume, got: %v", args)
	}
	if !containsPair(args, "--env", "POSTGRES_PASSWORD=secret") {
		t.Errorf("db run args missing env, got: %v", args)
	}
	if !containsPair(args, "--name", "basic-db-1") {
		t.Errorf("db run args missing name, got: %v", args)
	}
}

func containsPair(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}
