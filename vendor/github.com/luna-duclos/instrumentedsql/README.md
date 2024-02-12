[![GoDoc](https://godoc.org/github.com/luna-duclos/instrumentedsql?status.svg)](https://godoc.org/github.com/luna-duclos/instrumentedsql)

# instrumentedsql
A sql driver that will wrap any other driver and log/trace all its calls

## How to use

Please see the [documentation](https://godoc.org/github.com/luna-duclos/instrumentedsql) and [examples](https://github.com/luna-duclos/instrumentedsql/blob/master/sql_example_test.go)

## Go version support

The aim is to support all versions of Go starting at 1.9, when the various context methods we require to function were introduced
Go 1.8 is unfortunately not supported due to lack of support in Google's tracing package, though it can probably be made to work.

The build_all.sh script uses GVM to load every version of go and verify that the library builds and passes its tests.

## Roadmap

The project is largely in maintenance mode, new contributions and bugfixes are welcomed and reviewed promptly, but it is largely considered done and ready for production usage.
When new versions of Go are released, any new driver interfaces can be added, PRs are welcomed for this.
 
## Contributing

PRs and issues are welcomed and will be responded to in a timely fashion, please contribute!

Contributors will contain all people (not companies) that have contributed to the project.
LICENSE will list all copyright holders.
