// Package translate converts a resolved Docker Compose project model into the
// argument vectors understood by Apple's `container` CLI. It is the bridge
// between compose semantics and the container runtime.
package translate

// Docker Compose resource labels. fruitbox reuses the canonical
// `com.docker.compose.*` label namespace so that containers, networks and
// volumes it creates carry the same identifying metadata Docker Compose
// applies. This keeps tooling, inspection and project grouping compatible.
const (
	LabelProject         = "com.docker.compose.project"
	LabelService         = "com.docker.compose.service"
	LabelContainerNumber = "com.docker.compose.container-number"
	LabelOneoff          = "com.docker.compose.oneoff"
	LabelWorkingDir      = "com.docker.compose.project.working_dir"
	LabelConfigFiles     = "com.docker.compose.project.config_files"
	LabelVersion         = "com.docker.compose.version"
	LabelNetwork         = "com.docker.compose.network"
	LabelVolume          = "com.docker.compose.volume"
	// LabelConfigHash records a stable hash of the service's resolved config so
	// `up` can detect when a service changed and needs recreating.
	LabelConfigHash = "com.docker.compose.config-hash"
)

// fruitbox-native labels mirror the com.docker.compose.* set under the
// io.fruitbox.* namespace. Carrying both means resources created by fruitbox
// stay identifiable and manageable without depending on the Docker label
// scheme — a path out of the Docker ecosystem.
const (
	FBLabelProject         = "io.fruitbox.project"
	FBLabelService         = "io.fruitbox.service"
	FBLabelContainerNumber = "io.fruitbox.container-number"
	FBLabelOneoff          = "io.fruitbox.oneoff"
	FBLabelVersion         = "io.fruitbox.version"
	FBLabelNetwork         = "io.fruitbox.network"
	FBLabelVolume          = "io.fruitbox.volume"
	FBLabelConfigHash      = "io.fruitbox.config-hash"
)

// Version is the fruitbox version reported in resource labels.
const Version = "0.1.0"
