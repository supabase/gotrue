TAGS ?= sqlite
GO_BIN ?= go

install:
	$(GO_BIN) install -tags ${TAGS} -v .

tidy:
	$(GO_BIN) mod tidy

build:
	$(GO_BIN) build -v .

test:
	./test.sh -cover -v

lint:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run --enable-all

update:
	rm go.*
	$(GO_BIN) mod init github.com/gobuffalo/fizz
	$(GO_BIN) mod tidy --go=1.16
