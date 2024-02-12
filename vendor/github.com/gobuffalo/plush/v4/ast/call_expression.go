package ast

import (
	"bytes"
	"strings"
)

type CallExpression struct {
	TokenAble
	Callee    Expression
	Function  Expression
	Arguments []Expression
	Block     *BlockStatement
	ElseBlock *BlockStatement
}

var _ Comparable = &CallExpression{}
var _ Expression = &CallExpression{}

func (ce *CallExpression) validIfCondition() bool { return true }

func (ce *CallExpression) expressionNode() {}

func (ce *CallExpression) String() string {
	var out bytes.Buffer
	args := []string{}

	for _, a := range ce.Arguments {
		if a != nil {
			args = append(args, a.String())
		}
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	if ce.Block != nil {
		out.WriteString(" {\n")
		out.WriteString(ce.Block.String())
		out.WriteString("}")
	}

	if ce.ElseBlock != nil {
		out.WriteString(" else { ")
		out.WriteString(ce.ElseBlock.String())
		out.WriteString(" }")
	}

	return out.String()
}
