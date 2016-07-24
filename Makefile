BINARY=marill
.DEFAULT_GOAL: $(BINARY)

# add tests to bindata.go for inclusion
GOBINDATA=$(go-bindata tests/...)

SOURCEDIR=$(PWD)
SOURCES := $(shell find "$(SOURCEDIR)" -mindepth 1 -maxdepth 1 -name "*.go")

$(BINARY): $(SOURCES)
	go build -o ${BINARY}
