package translate

import (
	"strings"
	"testing"
)

func TestSecretsAndConfigsMount(t *testing.T) {
	proj := loadProject(t, "secrets")
	app, _ := proj.GetService("app")
	args, err := BuildRunArgs(proj, app, RunOptions{Number: 1, Detach: true})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")

	// Default secret target: /run/secrets/db_password, read-only.
	if !hasVolumeWithSuffix(args, "db_password.txt:/run/secrets/db_password:ro") {
		t.Errorf("missing default secret mount, args: %v", args)
	}
	// Explicit absolute target preserved.
	if !hasVolumeWithSuffix(args, "db_password.txt:/custom/secret/path:ro") {
		t.Errorf("missing custom-target secret mount, args: %v", args)
	}
	// Config default target: /app_config.
	if !hasVolumeWithSuffix(args, "app_config.yml:/app_config:ro") {
		t.Errorf("missing config mount, args: %v", args)
	}
	_ = joined
}

func hasVolumeWithSuffix(args []string, suffix string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--volume" && strings.HasSuffix(args[i+1], suffix) {
			return true
		}
	}
	return false
}
