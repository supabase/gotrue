package ast

import (
	"bytes"
)

type IndexExpression struct {
	TokenAble
	Left   Expression
	Index  Expression
	Value  Expression
	Callee Expression
}

var _ Comparable = &IndexExpression{}
var _ Expression = &IndexExpression{}

func (ie *IndexExpression) validIfCondition() bool { return true }

func (ie *IndexExpression) expressionNode() {}

func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())

	if ie.Callee != nil {
		out.WriteString("]")
		out.WriteString("." + ie.Callee.String())
		out.WriteString(")")
	} else {

		out.WriteString("])")
	}

	if ie.Value != nil {
		out.WriteString("=")
		out.WriteString(ie.Value.String())
	}

	return out.String()
}
