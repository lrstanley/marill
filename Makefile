.DEFAULT_GOAL := all

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

BINARY=marill
LD_FLAGS += -s -w

generate:
	@echo -e "\n\033[0;36m [ Generating gocode from assets... ]\033[0;m"
	test -f $(GOPATH)/bin/go-bindata || go get -v github.com/jteeuwen/go-bindata/...
	$(GOPATH)/bin/go-bindata data/...

fetch:
	@echo -e "\n\033[0;36m [ Fetching dependencies ]\033[0;m"
	# go get -v -d ./... <-- legacy style
	test -f $(GOPATH)/bin/govendor || go get -v -u github.com/kardianos/govendor
	test -f $(GOPATH)/bin/goreleaser || go get -u -v github.com/goreleaser/goreleaser

	$(GOPATH)/bin/govendor sync

update-deps: fetch
	@echo -e "\n\033[0;36m [ Updating dependencies ]\033[0;m"
	$(GOPATH)/bin/govendor add +external
	$(GOPATH)/bin/govendor remove +unused
	$(GOPATH)/bin/govendor update +external

release: fetch generate
	$(GOPATH)/bin/goreleaser --skip-publish

publish: fetch generate
	$(GOPATH)/bin/goreleaser

snapshot: fetch generate
	$(GOPATH)/bin/goreleaser --snapshot --skip-validate --skip-publish

lint: clean fetch generate
	@echo -e "\n\033[0;36m [ Installng linters ]\033[0;m"
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -v -u gopkg.in/alecthomas/gometalinter.v1
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	@echo -e "\n\033[0;36m [ Running SHORT linting ]\033[0;m"
	$(GOPATH)/bin/gometalinter.v1 --vendored-linters --sort=path --exclude="bindata*" --exclude "vendor" --min-confidence=0.3 --dupl-threshold=70 --deadline 15s --disable-all -E structcheck -E ineffassign -E dupl -E golint -E gotype -E varcheck -E interfacer -E goconst -E gosimple -E staticcheck -E unused -E gofmt -E goimports -E misspell ./...

lintextended: clean fetch generate
	@echo -e "\n\033[0;36m [ Installng linters ]\033[0;m"
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -v -u gopkg.in/alecthomas/gometalinter.v1
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	@echo -e "\n\033[0;36m [ Running EXTENDED linting ]\033[0;m"
	$(GOPATH)/bin/gometalinter.v1 --vendored-linters --sort=path --exclude="bindata*" --exclude "vendor" --min-confidence=0.3 --dupl-threshold=70 --deadline 1m --disable-all -E structcheck -E aligncheck -E ineffassign -E dupl -E golint -E gotype -E errcheck -E varcheck -E interfacer -E goconst -E gosimple -E staticcheck -E unused -E gofmt -E goimports -E misspell ./...

test: clean fetch generate
	@echo -e "\n\033[0;36m [ Running SHORT tests ]\033[0;m"
	go test -v -timeout 30s -short $(shell go list ./... | grep -v "vendor/")

testextended: clean fetch generate
	@echo -e "\n\033[0;36m [ Running EXTENDED tests ]\033[0;m"
	go test -v -timeout 2m $(shell go list ./... | grep -v "vendor/")

debug: clean fetch generate
	@echo -e "\n\033[0;36m [ Executing ]\033[0;m"
	go run -ldflags "${LD_FLAGS}" *.go -d

run: all
	@echo -e "\n\033[0;36m [ Executing ]\033[0;m"
	${BINARY}

clean:
	@echo -e "\n\033[0;36m [ Removing previously compiled binaries, and cleaning up ]\033[0;m"
	/bin/rm -vrf "${BINARY}" dist bindata.go

compress:
	@echo -e "\n\033[0;36m [ Attempting to compress with UPX ]\033[0;m"
	(which upx > /dev/null && upx --best -q dist/marill*/marill* > /dev/null) || echo "not using upx"

all: clean fetch generate
	@echo -e "\n\033[0;36m [ Removing previously compiled binaries ]\033[0;m"
	rm -vf ${BINARY}

	@echo -e "\n\033[0;36m [ Building ${BINARY} ]\033[0;m"
	go build -ldflags "${LD_FLAGS}" -x -v -o ${BINARY}
