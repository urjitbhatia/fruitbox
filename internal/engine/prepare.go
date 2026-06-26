package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// stateDir returns the directory where fruitbox writes generated per-container
// files (e.g. /etc/hosts, /etc/hostname). Defaults to a temp directory.
func (e *Engine) stateDir() string {
	if e.StateDir != "" {
		return e.StateDir
	}
	return filepath.Join(os.TempDir(), "fruitbox")
}

// prepareGeneratedMounts writes any generated config files a service needs
// (because the runtime can't set them via flags) and returns the corresponding
// read-only `--volume` specs to inject into the run command.
func (e *Engine) prepareGeneratedMounts(p *types.Project, svc types.ServiceConfig, number int) ([]string, error) {
	var mounts []string

	files := map[string]string{}
	if c := translate.HostnameFileContent(svc); c != "" {
		files["/etc/hostname"] = c
	}
	if c := translate.HostsFileContent(svc); c != "" {
		files["/etc/hosts"] = c
	}
	if len(files) == 0 {
		return nil, nil
	}

	dir := filepath.Join(e.stateDir(), p.Name, containerName(p, svc, number))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	// Deterministic mount order: by target path.
	for _, target := range sortedStringKeys(files) {
		base := filepath.Base(target) // hostname, hosts
		hostPath := filepath.Join(dir, base)
		if err := os.WriteFile(hostPath, []byte(files[target]), 0o644); err != nil {
			return nil, err
		}
		mounts = append(mounts, hostPath+":"+target+":ro")
	}
	return mounts, nil
}

// applySysctls runs the post-start sysctl exec commands for a service, if any.
func (e *Engine) applySysctls(ctx context.Context, p *types.Project, svc types.ServiceConfig, number int) {
	cname := containerName(p, svc, number)
	for _, args := range translate.SysctlExecArgs(cname, svc) {
		if _, err := e.Runner.Run(ctx, args...); err != nil {
			e.logf("WARNING: %s: failed to apply sysctl (%v)", svc.Name, err)
		}
	}
}

func sortedStringKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// small, stable
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
