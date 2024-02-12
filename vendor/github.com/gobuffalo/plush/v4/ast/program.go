package ast

import (
	"bytes"
	"strings"
)

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// InnerText gets the raw string representation of the program's contents.
func (p *Program) InnerText() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String() + "\n")
	}

	return strings.TrimRight(out.String(), "\n")
}
