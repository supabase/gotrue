github_flavored_markdown
========================

[![Standard Test](https://github.com/gobuffalo/github_flavored_markdown/actions/workflows/standard-go-test.yml/badge.svg)](https://github.com/gobuffalo/github_flavored_markdown/actions/workflows/standard-go-test.yml)
[![GoDoc](https://godoc.org/github.com/gobuffalo/github_flavored_markdown?status.svg)](https://godoc.org/github.com/gobuffalo/github_flavored_markdown)

Package github_flavored_markdown provides a GitHub Flavored Markdown renderer
with fenced code block highlighting, clickable heading anchor links.

The functionality should be equivalent to the GitHub Markdown API endpoint specified at
https://developer.github.com/v3/markdown/#render-a-markdown-document-in-raw-mode, except
the rendering is performed locally.

See examples for how to generate a complete HTML page, including CSS styles.

Installation
------------

```bash
go get github.com/gobuffalo/github_flavored_markdown
```

Directories
-----------

| Path                                                                                | Synopsis                                                                     |
|-------------------------------------------------------------------------------------|------------------------------------------------------------------------------|
| [gfmstyle](https://pkg.go.dev/github.com/gobuffalo/github_flavored_markdown/gfmstyle) | Package gfmstyle contains CSS styles for rendering GitHub Flavored Markdown. |

License
-------

- [MIT License](LICENSE)
