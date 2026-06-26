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
)

// Version is the fruitbox version reported in resource labels.
const Version = "0.1.0"
