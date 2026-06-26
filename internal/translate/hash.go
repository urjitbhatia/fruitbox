package translate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// ServiceConfigHash returns a stable hex hash of a service's resolved
// configuration. `up` records it as a container label and compares it on a
// later run to decide whether the service changed and must be recreated.
func ServiceConfigHash(svc types.ServiceConfig) string {
	b, err := json.Marshal(svc)
	if err != nil {
		// A non-marshalable service is unexpected; fall back to the name so the
		// hash is at least stable per service.
		return fmt.Sprintf("%x", sha256.Sum256([]byte(svc.Name)))
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
