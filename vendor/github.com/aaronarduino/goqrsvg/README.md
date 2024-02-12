# goqrsvg
goqrsvg is an API that makes QR Code to SVG conversions.

## To use:

`import "github.com/aaronarduino/goqrsvg"`

## Uses:
```
"github.com/ajstarks/svgo"
"github.com/boombuler/barcode/qr"
```

## Example Usage:

```
package main

import (
  "log"
  "net/http"

  "github.com/aaronarduino/goqrsvg"
  "github.com/ajstarks/svgo"
  "github.com/boombuler/barcode/qr"
)

func main() {
  http.Handle("/", http.HandlerFunc(barcodeCreate))
  err := http.ListenAndServe(":2003", nil)
  if err != nil {
    log.Fatal("ListenAndServe:", err)
  }
}

func barcodeCreate(w http.ResponseWriter, req *http.Request) {
  w.Header().Set("Content-Type", "image/svg+xml")
  s := svg.New(w)

  // Create the barcode
  qrCode, _ := qr.Encode("Hello World", qr.M, qr.Auto)

  // Write QR code to SVG
  qs := goqrsvg.NewQrSVG(qrCode, 5)
  qs.StartQrSVG(s)
  qs.WriteQrSVG(s)

  s.End()
}
```

## Documentation:
See [GoDoc](https://godoc.org/github.com/aaronarduino/goqrsvg)
