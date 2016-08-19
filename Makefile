BINARY=marill

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

.DEFAULT_GOAL := all

fetch:
	go get -u golang.org/x/net/html
	go get -u github.com/jteeuwen/go-bindata/...
	# add tests to bindata.go for inclusion
	go-bindata tests/...

test: fetch
	go test ./...

all: fetch
	go build -x -v -o ${BINARY}
	/bin/rm -fv bindata.go
