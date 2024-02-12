/*
Package fizz is a common DSL for writing SQL migrations
*/
package fizz

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	shellquote "github.com/kballard/go-shellquote"
)

// Options is a generic map of options.
type Options map[string]interface{}

type fizzer struct {
	Bubbler *Bubbler
}

func (f fizzer) add(s string, err error) error {
	if err != nil {
		return err
	}
	f.Bubbler.data = append(f.Bubbler.data, s)
	return nil
}

func (f fizzer) Exec(out io.Writer) func(string) error {
	return func(s string) error {
		args, err := shellquote.Split(s)
		if err != nil {
			return err
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = out
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return err
		}
		return nil
	}
}

// AFile reads in a fizz migration from an io.Reader and translates its contents to SQL.
func AFile(f io.Reader, t Translator) (string, error) {
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return AString(string(b), t)
}

// AString reads a fizz string, and translates its contents to SQL.
func AString(s string, t Translator) (string, error) {
	b := NewBubbler(t)
	return b.Bubble(s)
}
