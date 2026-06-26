.PHONY: build test test-integration compat fmt vet

# Build the fruitbox binary.
build:
	go build -o bin/fruitbox ./cmd/fruitbox

# Fast, hermetic unit tests (no container runtime needed).
test:
	go test ./...

# Integration tests that drive the real Apple `container` runtime.
# Requires macOS on Apple silicon with `container` installed and running.
# These skip automatically when the runtime is unavailable.
test-integration:
	go test -tags=integration -timeout 600s ./test/integration/...

# Differential compatibility audit vs. the locally installed `docker compose`.
compat: build
	./scripts/compat-audit.sh ./bin/fruitbox

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...
