OUT := cleanid3

APP_ID := tillt.$(OUT)
VERSION := $(shell git describe --always --long --dirty)
TIME := $(shell date)
DESTINATION := /usr/local
LDFLAGS := -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildTime=${TIME}'"
PACKAGE_DIR := _package

all: build

install: build
	@go install ${LDFLAGS}

build:
	go build -v -o ${OUT} ${LDFLAGS}

clean:
	-@rm -f ${OUT}
	-@rm -rf ${PACKAGE_DIR}

package: build
	mkdir -p ${PACKAGE_DIR}/share/${OUT}
	cp forbidden.txt ${PACKAGE_DIR}/share/${OUT}
	mkdir -p ${PACKAGE_DIR}/bin
	cp ${OUT} ${PACKAGE_DIR}/bin
	pkgbuild --root ${PACKAGE_DIR}	           	\
		--identifier ${APP_ID}                 		\
		--version ${VERSION}               				\
		--install-location ${DESTINATION}					\
		./${OUT}.pkg

lint:
	golint ./...

.PHONY: install build clean package lint
