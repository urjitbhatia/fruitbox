package translate

import (
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
)

// BuildImageTag returns the image tag a service's build output should be tagged
// with. When the service sets `image:`, that is used; otherwise Compose
// convention names it "<project>_<service>".
func BuildImageTag(projectName string, svc types.ServiceConfig) string {
	if svc.Image != "" {
		return svc.Image
	}
	return sanitize(projectName) + "_" + sanitize(svc.Name)
}

// BuildBuildArgs renders the arguments for `container build` from a service's
// build configuration. It returns nil if the service has no build section.
func BuildBuildArgs(projectName string, svc types.ServiceConfig) []string {
	b := svc.Build
	if b == nil {
		return nil
	}
	args := []string{"build"}

	args = append(args, "--tag", BuildImageTag(projectName, svc))

	if b.Dockerfile != "" {
		args = append(args, "--file", b.Dockerfile)
	}
	if b.Target != "" {
		args = append(args, "--target", b.Target)
	}
	if b.NoCache {
		args = append(args, "--no-cache")
	}
	if b.Pull {
		args = append(args, "--pull")
	}

	// Build args, sorted for determinism.
	for _, k := range sortedKeys(b.Args) {
		if v := b.Args[k]; v != nil {
			args = append(args, "--build-arg", k+"="+*v)
		} else {
			args = append(args, "--build-arg", k)
		}
	}

	// Build-time labels.
	for _, l := range sortedLabels(b.Labels) {
		args = append(args, "--label", l)
	}

	// Context is the trailing positional argument; default to ".".
	context := b.Context
	if context == "" {
		context = "."
	}
	args = append(args, context)
	return args
}

func sortedLabels(labels types.Labels) []string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+labels[k])
	}
	return out
}
