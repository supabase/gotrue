package ast

import (
	"bytes"
)

// BlockStatement is a list of statements grouped in a context surrounded by braces.
type BlockStatement struct {
	TokenAble
	Statements []Statement
}

var _ Statement = &BlockStatement{}

func (bs *BlockStatement) statementNode() {}

// InnerText gets the raw string representation of the block's contents.
func (bs *BlockStatement) InnerText() string {
	var out bytes.Buffer
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	for _, s := range bs.Statements {
		out.WriteString("\t" + s.String() + "\n")
	}
	return out.String()
}
