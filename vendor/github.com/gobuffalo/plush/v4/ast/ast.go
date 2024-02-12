package ast

import "github.com/gobuffalo/plush/v4/token"

type TokenAble struct {
	token.Token
}

func (t TokenAble) T() token.Token {
	return t.Token
}

func (t TokenAble) TokenLiteral() string {
	return t.Token.Literal
}

type Printable interface {
	Printable() bool
}

// The base Node interface
type Node interface {
	T() token.Token
	TokenLiteral() string
	String() string
}

// All statement nodes implement this
type Statement interface {
	Node
	statementNode()
}

// All expression nodes implement this
type Expression interface {
	Node
	expressionNode()
}

type Comparable interface {
	// TODO: not sure what is the purpose of this interface.
	// The only method of this interface is validIfCondition that returns
	// true always for all implementations. Need to check but it could be
	// something like isCondition or isComparable of Expression interface.
	validIfCondition() bool
}
