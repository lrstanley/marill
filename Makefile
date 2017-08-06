.DEFAULT_GOAL := all

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

BINARY=marill
LD_FLAGS += -s -w

generate:
	$(GOPATH)/bin/go-bindata data/...

fetch:
	test -f $(GOPATH)/bin/govendor || go get -v -u github.com/kardianos/govendor
	test -f $(GOPATH)/bin/goreleaser || go get -v -u github.com/goreleaser/goreleaser
	test -f $(GOPATH)/bin/go-bindata || go get -v -u github.com/jteeuwen/go-bindata/...

	$(GOPATH)/bin/govendor sync

update-deps: fetch
	$(GOPATH)/bin/govendor add +external
	$(GOPATH)/bin/govendor remove +unused
	$(GOPATH)/bin/govendor update +external

release: clean fetch generate
	$(GOPATH)/bin/goreleaser --skip-publish

publish: clean fetch generate
	$(GOPATH)/bin/goreleaser

snapshot: clean fetch generate
	$(GOPATH)/bin/goreleaser --snapshot --skip-validate --skip-publish

lint: clean fetch generate
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -v -u gopkg.in/alecthomas/gometalinter.v1
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	$(GOPATH)/bin/gometalinter.v1 --vendored-linters --sort=path --exclude="bindata*" --exclude "vendor" --min-confidence=0.3 --dupl-threshold=70 --deadline 15s --disable-all -E structcheck -E ineffassign -E dupl -E golint -E gotype -E varcheck -E interfacer -E goconst -E gosimple -E staticcheck -E unused -E gofmt -E goimports -E misspell ./...

lintextended: clean fetch generate
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -v -u gopkg.in/alecthomas/gometalinter.v1
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	$(GOPATH)/bin/gometalinter.v1 --vendored-linters --sort=path --exclude="bindata*" --exclude "vendor" --min-confidence=0.3 --dupl-threshold=70 --deadline 1m --disable-all -E structcheck -E aligncheck -E ineffassign -E dupl -E golint -E gotype -E errcheck -E varcheck -E interfacer -E goconst -E gosimple -E staticcheck -E unused -E gofmt -E goimports -E misspell ./...

test: clean fetch generate
	go test -v -timeout 30s -short $(shell go list ./... | grep -v "vendor/")

testextended: clean fetch generate
	go test -v -timeout 2m $(shell go list ./... | grep -v "vendor/")

clean:
	/bin/rm -vrf "${BINARY}" dist bindata.go

compress:
	(which /usr/bin/upx > /dev/null && find dist/*/* | xargs -I{} -n1 -P 4 /usr/bin/upx --best "{}") || echo "not using upx for binary compression"

all: clean fetch generate
	go build -ldflags "${LD_FLAGS}" -x -v -o ${BINARY}
