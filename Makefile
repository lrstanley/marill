BINARY=marill

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

.DEFAULT_GOAL := all

fetch:
	go get -u golang.org/x/net/html
	go get -u github.com/urfave/cli
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
	go build -x -v -o ${BINARY}
	/bin/rm -fv bindata.go
