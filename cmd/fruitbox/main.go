// Command fruitbox is a Docker Compose-compatible orchestrator for Apple's
// native `container` runtime.
package main

import (
	"os"

	"github.com/urjitbhatia/fruitbox/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
