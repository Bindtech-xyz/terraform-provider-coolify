# Local development workflow. `make install` + the dev_overrides block below is
# the fastest loop: no terraform init, no lock file, no version bump needed.
#
#   # ~/.terraformrc
#   provider_installation {
#     dev_overrides {
#       "bindtech-xyz/coolify" = "<your GOBIN, e.g. ~/go/bin>"
#     }
#     direct {}
#   }

default: install

.PHONY: build install lint fmt test testacc docs release

build:
	go build -v ./...

install:
	go install -v .

lint:
	golangci-lint run

fmt:
	gofmt -s -w .
	terraform fmt -recursive ./examples/

test:
	go test ./... -v -timeout=120s -parallel=4

# Acceptance tests create real objects on the Coolify instance pointed to by
# COOLIFY_ENDPOINT / COOLIFY_TOKEN. Use a disposable instance.
testacc:
	TF_ACC=1 go test ./internal/provider/ -v -timeout=30m

docs:
	go generate ./...

# make release VERSION=v0.1.1
release:
	@test -n "$(VERSION)" || { echo "usage: make release VERSION=vX.Y.Z" >&2; exit 1; }
	scripts/release.sh $(VERSION)
