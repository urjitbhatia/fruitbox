package translate

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// composeProject is an alias so callers (and tests) within this package can
// refer to the compose project model without importing the types package.
type composeProject = types.Project

var nameSanitizer = regexp.MustCompile(`[^a-z0-9_-]+`)

// ContainerName returns the canonical container name for a service replica,
// following Docker Compose v2 convention: "<project>-<service>-<number>".
// The project and service segments are lower-cased and stripped of characters
// that are not valid in a container name.
func ContainerName(project, service string, number int) string {
	p := sanitize(project)
	s := sanitize(service)
	return fmt.Sprintf("%s-%s-%d", p, s, number)
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	return nameSanitizer.ReplaceAllString(s, "")
}

// ContainerNameSegment lower-cases and sanitizes a string into a valid segment
// of a container name (project or service component).
func ContainerNameSegment(s string) string {
	return sanitize(s)
}

// RunOptions controls how run/create arguments are generated for a replica.
type RunOptions struct {
	// Number is the 1-based replica index of this container.
	Number int
	// Detach adds --detach so the container runs in the background.
	Detach bool
	// Remove adds --rm so the container is deleted on exit (one-off runs).
	Remove bool
	// Oneoff marks the container as a one-off (compose run) container.
	Oneoff bool
	// Create generates `create` arguments instead of `run`.
	Create bool
}

// BuildRunArgs converts a resolved compose service into the argument vector for
// `container run` (or `container create`). The returned slice begins with the
// subcommand and ends with the image and command, ready to be passed to the
// container CLI.
func BuildRunArgs(p *composeProject, svc types.ServiceConfig, opts RunOptions) ([]string, error) {
	image := svc.Image
	if image == "" && svc.Build != nil {
		image = BuildImageTag(p.Name, svc)
	}
	if image == "" {
		return nil, fmt.Errorf("service %q has neither image nor build", svc.Name)
	}

	verb := "run"
	if opts.Create {
		verb = "create"
	}
	args := []string{verb}

	name := svc.ContainerName
	if name == "" {
		name = ContainerName(p.Name, svc.Name, opts.Number)
	}
	args = append(args, "--name", name)

	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.Remove {
		args = append(args, "--rm")
	}

	// Resource limits.
	if svc.CPUS > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(float64(svc.CPUS), 'f', -1, 32))
	}
	if svc.MemLimit > 0 {
		args = append(args, "--memory", strconv.FormatInt(int64(svc.MemLimit), 10))
	}

	// Process configuration.
	if svc.User != "" {
		args = append(args, "--user", svc.User)
	}
	if svc.WorkingDir != "" {
		args = append(args, "--workdir", svc.WorkingDir)
	}
	if svc.ReadOnly {
		args = append(args, "--read-only")
	}
	if svc.Init != nil && *svc.Init {
		args = append(args, "--init")
	}
	if svc.ShmSize > 0 {
		args = append(args, "--shm-size", strconv.FormatInt(int64(svc.ShmSize), 10))
	}

	// entrypoint: container's --entrypoint takes a single executable; any
	// extra elements become leading command arguments (matching how compose
	// flattens an exec-form entrypoint).
	var entrypointRest []string
	if len(svc.Entrypoint) > 0 {
		args = append(args, "--entrypoint", svc.Entrypoint[0])
		entrypointRest = svc.Entrypoint[1:]
	}

	// Networks (sorted by service-local key for determinism).
	for _, netKey := range sortedKeys(svc.Networks) {
		args = append(args, "--network", networkName(p, netKey))
	}

	// Environment (sorted by key for determinism).
	for _, k := range sortedKeys(svc.Environment) {
		if v := svc.Environment[k]; v != nil {
			args = append(args, "--env", k+"="+*v)
		} else {
			args = append(args, "--env", k)
		}
	}

	// DNS.
	for _, d := range svc.DNS {
		args = append(args, "--dns", d)
	}
	for _, d := range svc.DNSSearch {
		args = append(args, "--dns-search", d)
	}
	for _, o := range svc.DNSOpts {
		args = append(args, "--dns-option", o)
	}

	// Capabilities.
	for _, c := range svc.CapAdd {
		args = append(args, "--cap-add", c)
	}
	for _, c := range svc.CapDrop {
		args = append(args, "--cap-drop", c)
	}

	// Ports.
	for _, port := range svc.Ports {
		args = append(args, "--publish", formatPort(port))
	}

	// Volumes and tmpfs.
	for _, vol := range svc.Volumes {
		flag, value, err := formatVolume(p, vol)
		if err != nil {
			return nil, err
		}
		args = append(args, flag, value)
	}
	for _, t := range svc.Tmpfs {
		args = append(args, "--tmpfs", t)
	}

	// Labels: compose identity labels plus user-defined labels, all sorted.
	for _, l := range buildLabels(p, svc, opts) {
		args = append(args, "--label", l)
	}

	// Image and command.
	args = append(args, image)
	args = append(args, entrypointRest...)
	args = append(args, svc.Command...)

	return args, nil
}

// buildLabels returns the sorted "key=value" label arguments for a container.
func buildLabels(p *composeProject, svc types.ServiceConfig, opts RunOptions) []string {
	labels := map[string]string{
		LabelProject:         p.Name,
		LabelService:         svc.Name,
		LabelContainerNumber: strconv.Itoa(opts.Number),
		LabelOneoff:          boolWord(opts.Oneoff),
	}
	for k, v := range svc.Labels {
		labels[k] = v
	}
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

func boolWord(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// networkName resolves a service-local network key to the actual network name
// the container runtime should attach to.
func networkName(p *composeProject, key string) string {
	if net, ok := p.Networks[key]; ok && net.Name != "" {
		return net.Name
	}
	return key
}

// formatPort renders a compose port mapping as container --publish syntax:
// [host-ip:][host-port:]container-port[/protocol].
func formatPort(port types.ServicePortConfig) string {
	var b strings.Builder
	if port.HostIP != "" {
		b.WriteString(port.HostIP)
		b.WriteString(":")
	}
	if port.Published != "" {
		b.WriteString(port.Published)
		b.WriteString(":")
	}
	b.WriteString(strconv.FormatUint(uint64(port.Target), 10))
	if port.Protocol != "" && port.Protocol != "tcp" {
		b.WriteString("/")
		b.WriteString(port.Protocol)
	}
	return b.String()
}

// formatVolume renders a compose volume mount as a container flag/value pair.
// Named volumes are resolved to their project-scoped runtime names.
func formatVolume(p *composeProject, vol types.ServiceVolumeConfig) (flag, value string, err error) {
	switch vol.Type {
	case types.VolumeTypeTmpfs:
		return "--tmpfs", vol.Target, nil
	case types.VolumeTypeBind, "":
		v := vol.Source + ":" + vol.Target
		if vol.ReadOnly {
			v += ":ro"
		}
		return "--volume", v, nil
	case types.VolumeTypeVolume:
		source := vol.Source
		if source == "" {
			// Anonymous volume: let the runtime allocate one.
			v := vol.Target
			return "--volume", v, nil
		}
		if named, ok := p.Volumes[source]; ok && named.Name != "" {
			source = named.Name
		}
		v := source + ":" + vol.Target
		if vol.ReadOnly {
			v += ":ro"
		}
		return "--volume", v, nil
	default:
		return "", "", fmt.Errorf("unsupported volume type %q on mount %q", vol.Type, vol.Target)
	}
}

// sortedKeys returns the keys of a string-keyed map in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
