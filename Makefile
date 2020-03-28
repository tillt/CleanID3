OUT := cleanid3

VERSION := $(shell git describe --always --long --dirty)
TIME := $(shell date)

LDFLAGS := -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildTime=${TIME}'"


all: build

install: build
	@go install ${LDFLAGS}

build:
	go build -i -v -o ${OUT} ${LDFLAGS}

clean:
	-@rm ${OUT} ${OUT}-v*

.PHONY: install static vet lint
