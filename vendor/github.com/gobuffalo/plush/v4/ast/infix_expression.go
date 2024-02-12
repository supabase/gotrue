package ast

import (
	"bytes"
)

type InfixExpression struct {
	TokenAble
	Left     Expression
	Operator string
	Right    Expression
}

var _ Comparable = &InfixExpression{}
var _ Expression = &InfixExpression{}

func (oe *InfixExpression) validIfCondition() bool { return true }

func (oe *InfixExpression) expressionNode() {}

func (oe *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")

	if oe.Left != nil {
		out.WriteString(oe.Left.String())
	}

	out.WriteString(" " + oe.Operator + " ")

	if oe.Right != nil {
		out.WriteString(oe.Right.String())
	} else {
		out.WriteString(" !!MISSING '%>'!!")
	}

	out.WriteString(")")

	return out.String()
}
