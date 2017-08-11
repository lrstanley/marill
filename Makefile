.DEFAULT_GOAL := build

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

BINARY=marill
LD_FLAGS += -s -w

help: ## Shows this help info.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

generate: ## Code generation.
	$(GOPATH)/bin/go-bindata data/...

fetch: ## Fetches the necessary dependencies to build.
	test -f $(GOPATH)/bin/govendor || go get -v -u github.com/kardianos/govendor
	test -f $(GOPATH)/bin/goreleaser || go get -v -u github.com/goreleaser/goreleaser
	test -f $(GOPATH)/bin/go-bindata || go get -v -u github.com/jteeuwen/go-bindata/...

	$(GOPATH)/bin/govendor sync

update-deps: fetch ## Updates all dependencies to the latest available versions.
	$(GOPATH)/bin/govendor add +external
	$(GOPATH)/bin/govendor remove +unused
	$(GOPATH)/bin/govendor update +external

snapshot: clean fetch generate ## Generate a snapshot release.
	$(GOPATH)/bin/goreleaser --snapshot --skip-validate --skip-publish

release: clean fetch generate ## Generate a release, but don't publish to GitHub.
	$(GOPATH)/bin/goreleaser --skip-validate --skip-publish

publish: clean fetch generate ## Generate a release, and publish to GitHub.
	$(GOPATH)/bin/goreleaser

lint: clean fetch generate ## Run linting.
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -v -u gopkg.in/alecthomas/gometalinter.v1
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	$(GOPATH)/bin/gometalinter.v1 --vendored-linters --sort=path --exclude="bindata*" --exclude "vendor" --min-confidence=0.3 --dupl-threshold=70 --deadline 15s --disable-all -E structcheck -E ineffassign -E dupl -E golint -E gotype -E varcheck -E interfacer -E goconst -E gosimple -E staticcheck -E unused -E gofmt -E goimports -E misspell ./...

lintextended: clean fetch generate ## Run extended linting (may take longer, more tests are run).
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -v -u gopkg.in/alecthomas/gometalinter.v1
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	$(GOPATH)/bin/gometalinter.v1 --vendored-linters --sort=path --exclude="bindata*" --exclude "vendor" --min-confidence=0.3 --dupl-threshold=70 --deadline 1m --disable-all -E structcheck -E aligncheck -E ineffassign -E dupl -E golint -E gotype -E errcheck -E varcheck -E interfacer -E goconst -E gosimple -E staticcheck -E unused -E gofmt -E goimports -E misspell ./...

test: clean fetch generate ## Runs builtin short tests.
	go test -v -timeout 30s -short $(shell go list ./... | grep -v "vendor/")

testextended: clean fetch generate ## Runs builtin tests.
	go test -v -timeout 2m $(shell go list ./... | grep -v "vendor/")

clean: ## Cleans up generated files/folders from the build.
	/bin/rm -vrf "${BINARY}" dist bindata.go

compress: ## Runs compression against the release-generated binaries using upx, if installed.
	(which /usr/bin/upx > /dev/null && find dist/*/* | xargs -I{} -n1 -P 4 /usr/bin/upx --best "{}") || echo "not using upx for binary compression"

build: clean fetch generate ## Multi-step build process.
	go build -ldflags "${LD_FLAGS}" -x -v -o ${BINARY}
