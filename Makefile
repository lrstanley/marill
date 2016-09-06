.DEFAULT_GOAL := all

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

BINARY=marill
VERSION=$(shell git describe --tags --abbrev=0 > /dev/null)
HASH=$(shell git rev-parse --short HEAD)
COMPILE_DATE=$(shell date -u '+%B %d %Y')
LD_FLAGS += -s -w -X 'main.version=$(VERSION)' -X 'main.commithash=$(HASH)' -X 'main.compiledate=$(COMPILE_DATE)'
RELEASE_ROOT = ./release

fetch:
	@echo "\n\033[0;36m [ Fetching dependencies ]\033[0;m"
	go get -d ./...
	which go-bindata > /dev/null || go get -u github.com/jteeuwen/go-bindata/...

lint: test
	@echo "\n\033[0;36m [ Verifying gometalinter is installed ]\033[0;m"
	test -f $(GOPATH)/bin/gometalinter.v1 || go get -u gopkg.in/alecthomas/gometalinter.v1

	@echo "\n\033[0;36m [ Installng linters ]\033[0;m"
	$(GOPATH)/bin/gometalinter.v1 -i > /dev/null
	@echo "\n\033[0;36m [ Running linters ]\033[0;m"
	$(GOPATH)/bin/gometalinter.v1 --cyclo-over=15 --min-confidence=.30 --deadline=10s --dupl-threshold=40 -E gofmt -E goimports -E misspell -E test ./...

test: fetch
	@echo "\n\033[0;36m [ Running tests ]\033[0;m"
	go test ./...

run: fetch
	@echo "\n\033[0;36m [ Executing ]\033[0;m"
	go run *.go

tools: fetch
	@echo "\n\033[0;36m [ Verifying dependencies are installed ]\033[0;m"
	go get github.com/mitchellh/gox

cc:
	@echo "\n\033[0;36m [ Cross compiling ]\033[0;m"
	gox -verbose -ldflags="${LD_FLAGS}" -os="linux freebsd netbsd openbsd" -arch="386 amd64 arm" -output "${RELEASE_ROOT}/pkg/{{.OS}}_{{.Arch}}/{{.Dir}}"

targz:
	@echo "\n\033[0;36m [ Compressing compiled binaries ]\033[0;m"
	mkdir -p ${RELEASE_ROOT}/dist
	cd ${RELEASE_ROOT}/pkg/;for osarch in *;do (cd $$osarch;tar zcvf ../../dist/marill_$$osarch.tar.gz ./* > /dev/null);done
	@echo "\n\033[0;36m [ Binaries compiled ]\033[0;m"
	find ${RELEASE_ROOT}/dist -type f

all: fetch
	@echo "\n\033[0;36m [ Removing previously compiled binaries ]\033[0;m"
	rm -vf ${BINARY}

	# add tests to bindata.go for inclusion
	go-bindata tests/...

	# using -ldflags "-s" is not fully supported, however it makes binary files much smaller. alternatively,
	#   - we could use -w, which just strips dwarf symbol tables, but -s makes things much smaller.
	#   - also note, this will make debugging with gdb nearly impossible.
	# 
	# using "-X 'var=value'" is supported in go 1.5+, and "-X 'var value'" is supported prior to that
	@echo "\n\033[0;36m [ Building ${BINARY} ]\033[0;m"
	go build -ldflags "${LD_FLAGS}" -x -v -o ${BINARY}
	/bin/rm -fv bindata.go

	@echo "\n\033[0;36m [ Attempting to compress ${BINARY} with UPX ]\033[0;m"
	(which upx > /dev/null && upx -1 -q ${BINARY} > /dev/null) || echo "not using upx"
	test -f ${BINARY}
