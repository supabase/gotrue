package pop

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/pop/v6/logging"
	"gopkg.in/yaml.v2"
)

// ErrConfigFileNotFound is returned when the pop config file can't be found,
// after looking for it.
var ErrConfigFileNotFound = errors.New("unable to find pop config file")

var lookupPaths = []string{"", "./config", "/config", "../", "../config", "../..", "../../config"}

// ConfigName is the name of the YAML databases config file
var ConfigName = "database.yml"

func init() {
	ap := os.Getenv("APP_PATH")
	if ap != "" {
		_ = AddLookupPaths(ap)
	}
	ap = os.Getenv("POP_PATH")
	if ap != "" {
		_ = AddLookupPaths(ap)
	}
}

// LoadConfigFile loads a POP config file from the configured lookup paths
func LoadConfigFile() error {
	path, err := findConfigPath()
	if err != nil {
		return err
	}
	Connections = map[string]*Connection{}
	log(logging.Debug, "Loading config file from %s", path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return LoadFrom(f)
}

// LookupPaths returns the current configuration lookup paths
func LookupPaths() []string {
	return lookupPaths
}

// AddLookupPaths add paths to the current lookup paths list
func AddLookupPaths(paths ...string) error {
	lookupPaths = append(paths, lookupPaths...)
	return nil
}

func findConfigPath() (string, error) {
	for _, p := range LookupPaths() {
		path, _ := filepath.Abs(filepath.Join(p, ConfigName))
		if _, err := os.Stat(path); err == nil {
			return path, err
		}
	}
	return "", ErrConfigFileNotFound
}

// LoadFrom reads a configuration from the reader and sets up the connections
func LoadFrom(r io.Reader) error {
	envy.Load()
	deets, err := ParseConfig(r)
	if err != nil {
		return err
	}
	for n, d := range deets {
		con, err := NewConnection(d)
		if err != nil {
			log(logging.Warn, "unable to load connection %s: %v", n, err)
			continue
		}
		Connections[n] = con
	}
	return nil
}

// ParseConfig reads the pop config from the given io.Reader and returns
// the parsed ConnectionDetails map.
func ParseConfig(r io.Reader) (map[string]*ConnectionDetails, error) {
	tmpl := template.New("test")
	tmpl.Funcs(map[string]interface{}{
		"envOr": func(s1, s2 string) string {
			return envy.Get(s1, s2)
		},
		"env": func(s1 string) string {
			return envy.Get(s1, "")
		},
	})
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	t, err := tmpl.Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("couldn't parse config template: %w", err)
	}

	var bb bytes.Buffer
	err = t.Execute(&bb, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't execute config template: %w", err)
	}

	deets := map[string]*ConnectionDetails{}
	err = yaml.Unmarshal(bb.Bytes(), &deets)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal config to yaml: %w", err)
	}
	return deets, nil
}
