package ast

import (
	"bytes"
	"strings"
)

type HashLiteral struct {
	TokenAble
	Order []Expression
	Pairs map[Expression]Expression
}

var _ Expression = &HashLiteral{}

func (hl *HashLiteral) expressionNode() {}

func (hl *HashLiteral) String() string {
	var out bytes.Buffer

	pairs := []string{}
	for _, key := range hl.Order {
		p := hl.Pairs[key]
		pairs = append(pairs, key.String()+": "+p.String())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}
