package translate

import "testing"

func TestBuildRunArgsExtras(t *testing.T) {
	proj := loadProject(t, "extras")
	app, _ := proj.GetService("app")
	args, err := BuildRunArgs(proj, app, RunOptions{Number: 1, Detach: true})
	if err != nil {
		t.Fatal(err)
	}
	if !containsPair(args, "--platform", "linux/arm64") {
		t.Errorf("missing platform: %v", args)
	}
	if !containsPair(args, "--runtime", "runc") {
		t.Errorf("missing runtime: %v", args)
	}
	if !containsPair(args, "--ulimit", "nofile=1024:4096") {
		t.Errorf("missing nofile ulimit: %v", args)
	}
	if !containsPair(args, "--ulimit", "nproc=512") {
		t.Errorf("missing nproc single ulimit: %v", args)
	}
}
