package translate

import (
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
)

// BuildNetworkCreateArgs renders the arguments for `container network create`
// for a compose network. It returns nil if the network is external (it must
// already exist and should not be created by fruitbox).
func BuildNetworkCreateArgs(projectName string, net types.NetworkConfig) []string {
	if net.External {
		return nil
	}
	args := []string{"network", "create"}

	if net.Internal {
		args = append(args, "--internal")
	}
	if ipam := firstSubnet(net); ipam != "" {
		args = append(args, "--subnet", ipam)
	}
	for _, k := range sortedKeys(net.DriverOpts) {
		args = append(args, "--option", k+"="+net.DriverOpts[k])
	}
	for _, l := range labelArgs(net.Labels, LabelProject, projectName, LabelNetwork, networkKeyName(net)) {
		args = append(args, "--label", l)
	}

	args = append(args, net.Name)
	return args
}

// BuildVolumeCreateArgs renders the arguments for `container volume create`
// for a compose volume. It returns nil for external volumes.
func BuildVolumeCreateArgs(projectName string, vol types.VolumeConfig) []string {
	if vol.External {
		return nil
	}
	args := []string{"volume", "create"}

	for _, k := range sortedKeys(vol.DriverOpts) {
		args = append(args, "--opt", k+"="+vol.DriverOpts[k])
	}
	for _, l := range labelArgs(vol.Labels, LabelProject, projectName, LabelVolume, vol.Name) {
		args = append(args, "--label", l)
	}

	args = append(args, vol.Name)
	return args
}

func firstSubnet(net types.NetworkConfig) string {
	for _, cfg := range net.Ipam.Config {
		if cfg.Subnet != "" {
			return cfg.Subnet
		}
	}
	return ""
}

func networkKeyName(net types.NetworkConfig) string {
	return net.Name
}

// labelArgs merges identity labels with user labels and returns them sorted as
// "key=value" strings. The variadic identity pairs are applied first and may be
// overridden by user labels.
func labelArgs(user types.Labels, identity ...string) []string {
	labels := map[string]string{}
	for i := 0; i+1 < len(identity); i += 2 {
		labels[identity[i]] = identity[i+1]
	}
	for k, v := range user {
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
