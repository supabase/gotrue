# X-Forwarded-For middleware fo Go [![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/sebest/xff) [![Build Status](https://travis-ci.org/sebest/xff.svg?branch=master)](https://travis-ci.org/sebest/xff)

Package `xff` is a `net/http` middleware/handler to parse [Forwarded HTTP Extension](http://tools.ietf.org/html/rfc7239) in Golang.

## Example usage

Install `xff`:

    go get github.com/sebest/xff

Edit `server.go`:

```go
package main

import (
  "net/http"

  "github.com/sebest/xff"
)

func main() {
  handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello from " + r.RemoteAddr + "\n"))
  })

  xffmw, _ := xff.Default()
  http.ListenAndServe(":8080", xffmw.Handler(handler))
}
```

Then run your server:

    go run server.go

The server now runs on `localhost:8080`:

    $ curl -D - -H 'X-Forwarded-For: 42.42.42.42' http://localhost:8080/
    HTTP/1.1 200 OK
    Date: Fri, 20 Feb 2015 20:03:02 GMT
    Content-Length: 29
    Content-Type: text/plain; charset=utf-8

    hello from 42.42.42.42:52661
