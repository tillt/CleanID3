OUT := cleanid3

APP_ID := tillt.$(OUT)
VERSION := $(shell git describe --always --long --dirty)
TIME := $(shell date)
DESTINATION := /usr/local
LDFLAGS := -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildTime=${TIME}'"
PACKAGE_DIR := _package
PACKAGE_ROOT := ${PACKAGE_DIR}${DESTINATION}
OS := $(shell uname)
CWD := $(shell pwd)

all: build

install: build
	mkdir -p $(DESTINATION)/share/$(OUT)
	ln -sfn $(CWD)/forbidden-words.txt $(DESTINATION)/share/$(OUT)/forbidden-words.txt
	ln -sfn $(CWD)/forbidden-bins.txt $(DESTINATION)/share/$(OUT)/forbidden-bins.txt
	go install $(LDFLAGS)

build:
	go build -v -o ${OUT} ${LDFLAGS}

clean:
	-@rm -f ${OUT}
	-@rm -rf ${PACKAGE_DIR}

archive: build
	mkdir -p ${PACKAGE_ROOT}/share/${OUT}
	cp forbidden-words.txt ${PACKAGE_ROOT}/share/${OUT}
	cp forbidden-bins.txt ${PACKAGE_ROOT}/share/${OUT}
	mkdir -p ${PACKAGE_ROOT}/bin
	cp ${OUT} ${PACKAGE_ROOT}/bin
ifeq ($(OS),Darwin)
	pkgbuild --root ${PACKAGE_ROOT}								\
		--identifier ${APP_ID}                 			\
		--version ${VERSION}               					\
		--install-location ${DESTINATION}						\
		./${OUT}.pkg
else
	cd ${PACKAGE_DIR} &&  tar czf ../${OUT}.tar.gz ./
endif

lint:
	golint ./...

.PHONY: install build clean archive lint
