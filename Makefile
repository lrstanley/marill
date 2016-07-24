BINARY=marill
SOURCEDIR=$(PWD)
SOURCES := $(shell find "$(SOURCEDIR)" -mindepth 1 -maxdepth 1 -name "*.go")

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build -o ${BINARY} ${SOURCES}
