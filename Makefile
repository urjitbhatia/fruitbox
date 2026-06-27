.PHONY: build test test-compat test-integration compat compat-matrix fmt vet

# Build the fruitbox binary.
build:
	go build -o bin/fruitbox ./cmd/fruitbox

# Fast, hermetic unit tests (no container runtime needed).
test:
	go test ./...

# Version-sensitive differential tests against a local `docker compose`
# (flag-parity ratchet + config oracle). Opt-in; excluded from the default
# suite because they depend on the installed docker compose version.
test-compat:
	FRUITBOX_COMPAT=1 go test ./internal/cli/...

# Integration tests that drive the real Apple `container` runtime.
# Requires macOS on Apple silicon with `container` installed and running.
# These skip automatically when the runtime is unavailable.
test-integration:
	go test -tags=integration -timeout 600s ./test/integration/...

# Differential compatibility audit vs. the locally installed `docker compose`.
compat: build
	./scripts/compat-audit.sh ./bin/fruitbox

# Flag-parity matrix across several pinned docker compose versions. Downloads
# release binaries (cached) and renders a markdown table. Requires gh + python3.
compat-matrix:
	./scripts/compat-matrix.sh

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...
