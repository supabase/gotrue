package ast

type ContinueExpression struct {
	TokenAble
}

var _ Expression = &ContinueExpression{}

func (ce *ContinueExpression) expressionNode() {}

func (ce *ContinueExpression) String() string {
	return ce.Token.Literal
}
