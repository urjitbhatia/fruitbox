package translate

import (
	"reflect"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestBuildNetworkCreateArgs(t *testing.T) {
	net := types.NetworkConfig{Name: "basic_net"}
	args := BuildNetworkCreateArgs("basic", net)
	want := []string{
		"network", "create",
		"--label", "com.docker.compose.network=basic_net",
		"--label", "com.docker.compose.project=basic",
		"basic_net",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("network args mismatch:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildNetworkCreateArgsExternalSkipped(t *testing.T) {
	net := types.NetworkConfig{Name: "ext", External: true}
	if args := BuildNetworkCreateArgs("basic", net); args != nil {
		t.Errorf("external network should be skipped, got %v", args)
	}
}

func TestBuildVolumeCreateArgs(t *testing.T) {
	vol := types.VolumeConfig{Name: "basic_dbdata"}
	args := BuildVolumeCreateArgs("basic", vol)
	want := []string{
		"volume", "create",
		"--label", "com.docker.compose.project=basic",
		"--label", "com.docker.compose.volume=basic_dbdata",
		"basic_dbdata",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("volume args mismatch:\n got: %v\nwant: %v", args, want)
	}
}
