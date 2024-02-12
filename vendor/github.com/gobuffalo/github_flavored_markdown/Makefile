TAGS ?= "sqlite"
GO_BIN ?= go

install:
	packr2
	$(GO_BIN) install -v .

deps:
	$(GO_BIN) get github.com/gobuffalo/release
	$(GO_BIN) get github.com/gobuffalo/packr/v2
	$(GO_BIN) get -tags ${TAGS} -t ./...
	$(GO_BIN) mod tidy

build:
	packr2
	$(GO_BIN) build -v .

test:
	packr2
	$(GO_BIN) test -tags ${TAGS} ./...

ci-test: deps
	$(GO_BIN) test -tags ${TAGS} -race ./...

lint:
	gometalinter --vendor ./... --deadline=1m --skip=internal

update:
	$(GO_BIN) get -u -tags ${TAGS}
	$(GO_BIN) mod tidy
	packr2
	make test
	make install
	$(GO_BIN) mod tidy

release-test:
	$(GO_BIN) test -tags ${TAGS} -race ./...

release:
	release -y -f version.go
