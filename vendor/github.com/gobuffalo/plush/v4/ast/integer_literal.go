package ast

type IntegerLiteral struct {
	TokenAble
	Value int
}

var _ Comparable = &IntegerLiteral{}
var _ Expression = &IntegerLiteral{}

func (il *IntegerLiteral) validIfCondition() bool { return true }

func (il *IntegerLiteral) expressionNode() {}

func (il *IntegerLiteral) String() string {
	return il.Token.Literal
}
