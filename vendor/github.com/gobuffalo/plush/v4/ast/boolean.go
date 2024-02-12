package ast

type Boolean struct {
	TokenAble
	Value bool
}

var _ Comparable = &Boolean{}
var _ Expression = &Boolean{}

func (b *Boolean) validIfCondition() bool { return true }

func (b *Boolean) expressionNode() {}

func (b *Boolean) String() string {
	return b.Token.Literal
}
