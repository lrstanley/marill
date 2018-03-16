.DEFAULT_GOAL := build

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

BINARY = marill
COMPRESS_CONC ?= $(shell nproc)
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null | sed -r "s:^v::g")
RSRC=README_TPL.md
ROUT=README.md

help: ## Shows this help info.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

readme-gen: ## Generates readme from template file
	cp -av "${RSRC}" "${ROUT}"
	sed -ri -e "s:\[\[tag\]\]:${VERSION}:g" -e "s:\[\[os\]\]:linux:g" -e "s:\[\[arch\]\]:amd64:g" "${ROUT}"

generate: readme-gen ## Code generation.
	$(GOPATH)/bin/go-bindata data/...

fetch: ## Fetches the necessary dependencies to build.
	test -f $(GOPATH)/bin/govendor || go get -v -u github.com/kardianos/govendor
	test -f $(GOPATH)/bin/goreleaser || go get -v -u github.com/goreleaser/goreleaser
	test -f $(GOPATH)/bin/go-bindata || go get -v -u github.com/jteeuwen/go-bindata/...

	$(GOPATH)/bin/govendor sync

update-deps: fetch ## Updates missing deps, removes unused, and updates based on local $GOPATH.
	$(GOPATH)/bin/govendor add -v +external
	$(GOPATH)/bin/govendor remove -v +unused
	$(GOPATH)/bin/govendor update -v +external

upgrade-deps: update-deps ## Upgrades all dependencies to the latest from origin.
	$(GOPATH)/bin/govendor fetch -v +vendor

snapshot: clean fetch generate ## Generate a snapshot release.
	$(GOPATH)/bin/goreleaser --snapshot --skip-validate --skip-publish

release: clean fetch generate ## Generate a release, but don't publish to GitHub.
	$(GOPATH)/bin/goreleaser --skip-validate --skip-publish

publish: clean fetch generate ## Generate a release, and publish to GitHub.
	$(GOPATH)/bin/goreleaser

test: clean fetch generate ## Runs builtin short tests.
	go test -v -timeout 30s -short $(shell go list ./... | grep -v "vendor/")

testextended: clean fetch generate ## Runs builtin tests.
	go test -v -timeout 2m $(shell go list ./... | grep -v "vendor/")

clean: ## Cleans up generated files/folders from the build.
	/bin/rm -vrf "${BINARY}" dist bindata.go

compress: ## Uses upx to compress release binaries (if installed, uses all cores/parallel comp.)
	(which upx > /dev/null && find dist/*/* | xargs -I{} -n1 -P ${COMPRESS_CONC} upx --best "{}") || echo "not using upx for binary compression"

build: clean fetch generate ## Multi-step build process.
	CGO_ENABLED=0 go build -ldflags '-d -s -w' -tags netgo -installsuffix netgo -v -o "${BINARY}"
