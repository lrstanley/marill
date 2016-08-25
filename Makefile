BINARY=marill

GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

.DEFAULT_GOAL := all

fetch:
	go get -u golang.org/x/net/html
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/golang/lint/golint

test: fetch
	go test ./...
	go vet ./...
	golint -set_exit_status=1 ./... 

all: fetch
	# add tests to bindata.go for inclusion
	go-bindata tests/...
	go build -x -v -o ${BINARY}
	/bin/rm -fv bindata.go
