package ast

type StringLiteral struct {
	TokenAble
	Value string
}

var _ Comparable = &StringLiteral{}
var _ Expression = &StringLiteral{}

func (sl *StringLiteral) validIfCondition() bool { return true }

func (sl *StringLiteral) expressionNode() {}

func (sl *StringLiteral) String() string {
	return "\"" + sl.Token.Literal + "\""
}
