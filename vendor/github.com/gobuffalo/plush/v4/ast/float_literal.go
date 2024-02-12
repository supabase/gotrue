package ast

type FloatLiteral struct {
	TokenAble
	Value float64
}

var _ Comparable = &FloatLiteral{}
var _ Expression = &FloatLiteral{}

func (il *FloatLiteral) validIfCondition() bool { return true }

func (il *FloatLiteral) expressionNode() {}

func (il *FloatLiteral) String() string {
	return il.Token.Literal
}
