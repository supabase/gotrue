package ast

import (
	"bytes"

	"github.com/gobuffalo/plush/v4/token"
)

type ReturnStatement struct {
	Type string
	TokenAble
	ReturnValue Expression
}

var _ Printable = &ReturnStatement{}
var _ Statement = &ReturnStatement{}

func (rs *ReturnStatement) Printable() bool {
	return true
}

func (rs *ReturnStatement) statementNode() {}

func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	if rs.Type == token.E_START {
		out.WriteString("<%= ")
	} else {
		out.WriteString("return ")
	}

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	if rs.Type == token.E_START {
		out.WriteString("; %>")
	} else {
		out.WriteString(";")
	}

	return out.String()
}
