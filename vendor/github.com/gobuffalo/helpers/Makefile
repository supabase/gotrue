TAGS ?= ""
GO_BIN ?= go

tidy:
	$(GO_BIN) mod tidy

test: 
	$(GO_BIN) test -cover -race -tags ${TAGS} ./...

lint:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run --enable-all

update:
	rm go.*
	$(GO_BIN) mod init github.com/gobuffalo/helpers
	$(GO_BIN) mod tidy -go=1.16

