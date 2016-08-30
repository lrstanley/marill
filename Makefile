.DEFAULT_GOAL := all

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

BINARY=marill
VERSION=$(shell git describe --tags --abbrev=0 > /dev/null)
HASH=$(shell git rev-parse --short HEAD)
COMPILE_DATE=$(shell date -u '+%B %d %Y')

fetch:
	go get -d ./...
	which go-bindata > /dev/null || go get -u github.com/jteeuwen/go-bindata/...

lint:
	which golint > /dev/null || go get -u github.com/golang/lint/golint
	go vet ./...
	golint -min_confidence=0.3 -set_exit_status=1 ./...

test: fetch
	go test ./...

all: fetch
	# add tests to bindata.go for inclusion
	go-bindata tests/...
	# using "-X 'var=value'" is supported in go 1.5+, and "-X 'var value'" is supported prior to that
	go build -ldflags "-X 'main.version=$(VERSION)' -X 'main.commithash=$(HASH)' -X 'main.compiledate=$(COMPILE_DATE)'" -x -v -o ${BINARY}
	/bin/rm -fv bindata.go
