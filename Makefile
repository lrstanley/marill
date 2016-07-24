BINARY=marill


GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

all:
	go get -u golang.org/x/net/html
	go get -u github.com/jteeuwen/go-bindata/...
	# add tests to bindata.go for inclusion
	go-bindata tests/...
	go build -v -o ${BINARY}
