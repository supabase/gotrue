package plush

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"sync"

	"github.com/gobuffalo/helpers/hctx"
)

// DefaultTimeFormat is the default way of formatting a time.Time type.
// This a **GLOBAL** variable, so if you change it, it will change for
// templates rendered through the `plush` package. If you want to set a
// specific time format for a particular call to `Render` you can set
// the `TIME_FORMAT` in the context.
//
/*
	ctx.Set("TIME_FORMAT", "2006-02-Jan")
	s, err = Render(input, ctx)
*/
var DefaultTimeFormat = "January 02, 2006 15:04:05 -0700"

var CacheEnabled bool
var cache = map[string]*Template{}
var moot = &sync.Mutex{}

// BuffaloRenderer implements the render.TemplateEngine interface allowing velvet to be used as a template engine
// for Buffalo
func BuffaloRenderer(input string, data map[string]interface{}, helpers map[string]interface{}) (string, error) {
	t, err := Parse(input)
	if err != nil {
		return "", err
	}

	for k, v := range helpers {
		data[k] = v
	}

	return t.Exec(NewContextWith(data))
}

// Parse an input string and return a Template, and caches the parsed template.
func Parse(input string) (*Template, error) {
	if !CacheEnabled {
		return NewTemplate(input)
	}

	moot.Lock()
	defer moot.Unlock()

	t, ok := cache[input]
	if ok {
		return t, nil
	}

	t, err := NewTemplate(input)
	if err != nil {
		return t, err
	}

	cache[input] = t
	return t, nil
}

// Render a string using the given the context.
func Render(input string, ctx hctx.Context) (string, error) {
	t, err := Parse(input)
	if err != nil {
		return "", err
	}
	return t.Exec(ctx)
}

func RenderR(input io.Reader, ctx hctx.Context) (string, error) {
	b, err := ioutil.ReadAll(input)
	if err != nil {
		return "", err
	}
	return Render(string(b), ctx)
}

// RunScript allows for "pure" plush scripts to be executed.
func RunScript(input string, ctx hctx.Context) error {
	input = "<% " + input + "%>"

	ctx = ctx.New()
	ctx.Set("print", func(i interface{}) {
		fmt.Print(i)
	})
	ctx.Set("println", func(i interface{}) {
		fmt.Println(i)
	})

	_, err := Render(input, ctx)
	return err
}

type interfaceable interface {
	Interface() interface{}
}

// HTMLer generates HTML source
type HTMLer interface {
	HTML() template.HTML
}
