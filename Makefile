# Veridian E-invoicing Pakistan — build automation.
#
#   make build                      host binary in dist/einvoice (uses the committed web UI)
#   make web                        rebuild the React UI into internal/webui/dist (needs Node.js 18+)
#   make release VERSION=1.2.0 LICENSE_PUBKEY=<base64>
#                                   Windows + Linux binaries in dist/<os>-<arch>/
#   make test / make vet / make clean
#
# LICENSE_PUBKEY is the public key printed by `go run ./cmd/licensegen keygen`.
# Builds without it run in developer mode (no licence enforcement) — never ship those.

VERSION        ?= 1.0.0
BUILD_DATE     ?= $(shell date -u +%Y-%m-%d)
LICENSE_PUBKEY ?=
GO             := GOTOOLCHAIN=local go
LDFLAGS        := -s -w \
	-X einvoicing/internal/brand.Version=$(VERSION) \
	-X einvoicing/internal/brand.BuildDate=$(BUILD_DATE) \
	-X einvoicing/internal/license.PublicKeyB64=$(LICENSE_PUBKEY)

.PHONY: all build web release test vet clean

all: build

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/einvoice ./cmd/einvoice

web:
	cd web && npm ci && npm run build

release:
	VERSION=$(VERSION) BUILD_DATE=$(BUILD_DATE) LICENSE_PUBKEY=$(LICENSE_PUBKEY) sh scripts/build-release.sh

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf dist
