package ast

type BreakExpression struct {
	TokenAble
}

var _ Expression = &BreakExpression{}

func (ce *BreakExpression) expressionNode() {}

func (ce *BreakExpression) String() string {
	return ce.Token.Literal
}
