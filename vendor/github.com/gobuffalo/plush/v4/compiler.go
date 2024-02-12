package plush

import (
	"bytes"
	"fmt"

	"github.com/gobuffalo/plush/v4/token"

	"html/template"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gobuffalo/helpers/hctx"
	"github.com/gobuffalo/plush/v4/ast"
)

type ErrUnknownIdentifier struct {
	ID  string
	Err error
}

func (e *ErrUnknownIdentifier) Error() string {
	if e.Err == nil {
		e.Err = fmt.Errorf("unknown identifier")
	}
	return fmt.Sprintf("%q: %s", e.ID, e.Err)
}

type compiler struct {
	ctx     hctx.Context
	program *ast.Program
	curStmt ast.Statement
	inCheck bool
}

func (c *compiler) compile() (string, error) {
	bb := &bytes.Buffer{}

	for _, stmt := range c.program.Statements {
		var res interface{}
		var err error

		switch node := stmt.(type) {
		case *ast.ReturnStatement:
			res, err = c.evalReturnStatement(node)

		case *ast.ExpressionStatement:
			if h, ok := node.Expression.(*ast.HTMLLiteral); ok {
				res = template.HTML(h.Value)
			} else {
				_, err = c.evalExpression(node.Expression)
			}
		case *ast.LetStatement:
			res, err = c.evalLetStatement(node)
		}

		if err != nil {
			s := stmt
			if c.curStmt != nil {
				s = c.curStmt
			}
			return "", fmt.Errorf("line %d: %w", s.T().LineNumber, err)
		}

		c.write(bb, res)
	}

	return bb.String(), nil
}

func (c *compiler) write(bb *bytes.Buffer, i interface{}) {
	switch t := i.(type) {
	case time.Time:
		if dtf, ok := c.ctx.Value("TIME_FORMAT").(string); ok {
			bb.WriteString(t.Format(dtf))
			return
		}
		bb.WriteString(t.Format(DefaultTimeFormat))
	case *time.Time:
		c.write(bb, *t)
	case interfaceable:
		c.write(bb, t.Interface())
	case string, ast.Printable, bool:
		bb.WriteString(template.HTMLEscaper(t))
	case template.HTML:
		bb.WriteString(string(t))
	case HTMLer:
		bb.WriteString(string(t.HTML()))
	case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64, float32, float64:
		bb.WriteString(fmt.Sprint(t))
	case fmt.Stringer:
		bb.WriteString(t.String())
	case []string:
		for _, ii := range t {
			c.write(bb, ii)
		}
	case []interface{}:
		for _, ii := range t {
			c.write(bb, ii)
		}
	case returnObject:
		for _, ii := range t.Value {
			c.write(bb, ii)
		}
	}
}

func (c *compiler) evalExpression(node ast.Expression) (interface{}, error) {
	switch s := node.(type) {
	case *ast.HTMLLiteral:
		return template.HTML(s.Value), nil
	case *ast.StringLiteral:
		return s.Value, nil
	case *ast.IntegerLiteral:
		return s.Value, nil
	case *ast.FloatLiteral:
		return s.Value, nil
	case *ast.InfixExpression:
		return c.evalInfixExpression(s)
	case *ast.HashLiteral:
		return c.evalHashLiteral(s)
	case *ast.IndexExpression:
		return c.evalIndexExpression(s)
	case *ast.CallExpression:
		return c.evalCallExpression(s)
	case *ast.Identifier:
		return c.evalIdentifier(s)
	case *ast.Boolean:
		return s.Value, nil
	case *ast.ArrayLiteral:
		return c.evalArrayLiteral(s)
	case *ast.ForExpression:
		return c.evalForExpression(s)
	case *ast.IfExpression:
		return c.evalIfExpression(s)
	case *ast.PrefixExpression:
		return c.evalPrefixExpression(s)
	case *ast.FunctionLiteral:
		return c.evalFunctionLiteral(s)
	case *ast.AssignExpression:
		return c.evalAssignExpression(s)
	case *ast.ContinueExpression:
		return continueObject{}, nil
	case *ast.BreakExpression:
		return breakObject{}, nil
	case nil:
		return nil, nil
	}
	return nil, fmt.Errorf("could not evaluate node %T", node)
}

func (c *compiler) evalAssignExpression(node *ast.AssignExpression) (interface{}, error) {
	v, err := c.evalExpression(node.Value)
	if err != nil {
		return nil, err
	}

	n := node.Name.Value
	if !c.ctx.Has(n) {
		return nil, &ErrUnknownIdentifier{
			ID: n,
		}
	}

	c.ctx.Set(n, v)
	return nil, nil
}

func (c *compiler) evalUserFunction(node *userFunction, args []ast.Expression) (interface{}, error) {
	octx := c.ctx
	defer func() { c.ctx = octx }()

	c.ctx = c.ctx.New()
	for i, p := range node.Parameters {
		a := args[i]
		v, err := c.evalExpression(a)
		if err != nil {
			return nil, err
		}

		c.ctx.Set(p.Value, v)
	}

	return c.evalBlockStatement(node.Block)
}

func (c *compiler) evalFunctionLiteral(node *ast.FunctionLiteral) (interface{}, error) {
	params := node.Parameters
	block := node.Block
	return &userFunction{Parameters: params, Block: block}, nil
}

func (c *compiler) evalPrefixExpression(node *ast.PrefixExpression) (interface{}, error) {
	res, err := c.evalExpression(node.Right)
	if err != nil {
		if _, ok := err.(*ErrUnknownIdentifier); !ok {
			return nil, err
		}
	}

	switch node.Operator {
	case "!":
		return !c.isTruthy(res), nil
	}

	return nil, fmt.Errorf("unknown operator %s", node.Operator)
}

func (c *compiler) evalIfExpression(node *ast.IfExpression) (interface{}, error) {
	con, err := c.evalExpression(node.Condition)
	if err != nil {
		if _, ok := err.(*ErrUnknownIdentifier); !ok {
			return nil, err
		}
	}

	if c.isTruthy(con) {
		return c.evalBlockStatement(node.Block)
	}

	return c.evalElseAndElseIfExpressions(node)
}

func (c *compiler) evalElseAndElseIfExpressions(node *ast.IfExpression) (interface{}, error) {
	var r interface{}
	for _, eiNode := range node.ElseIf {
		eiCon, err := c.evalExpression(eiNode.Condition)
		if err != nil {
			if _, ok := err.(*ErrUnknownIdentifier); !ok {
				return nil, err
			}
		}

		if c.isTruthy(eiCon) {
			return c.evalBlockStatement(eiNode.Block)
		}
	}

	if node.ElseBlock != nil {
		return c.evalBlockStatement(node.ElseBlock)
	}

	return r, nil
}

func (c *compiler) isTruthy(i interface{}) bool {
	if i == nil {
		return false
	}

	switch t := i.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case template.HTML:
		return t != ""
	default:
		if reflect.ValueOf(i).Kind() == reflect.Ptr && reflect.ValueOf(i).IsNil() {
			return false
		}

		return true
	}
}

func (c *compiler) evalIndexExpression(node *ast.IndexExpression) (interface{}, error) {
	index, err := c.evalExpression(node.Index)
	if err != nil {
		return nil, err
	}

	left, err := c.evalExpression(node.Left)
	if err != nil {
		return nil, err
	}

	var value interface{}

	if node.Value != nil {
		value, err = c.evalExpression(node.Value)
		if err != nil {
			return nil, err
		}

		return nil, c.evalUpdateIndex(left, index, value)
	}

	return c.evalAccessIndex(left, index, node)
}

func (c *compiler) evalUpdateIndex(left, index, value interface{}) error {
	var err error
	rv := reflect.ValueOf(left)
	switch rv.Kind() {
	case reflect.Map:
		rv.SetMapIndex(reflect.ValueOf(index), reflect.ValueOf(value))
	case reflect.Array, reflect.Slice:
		if i, ok := index.(int); ok {
			if rv.Len()-1 < i {
				err = fmt.Errorf("array index out of bounds, got index %d, while array size is %v", i, rv.Len())
			} else {
				rv.Index(i).Set(reflect.ValueOf(value))
			}
		} else {
			err = fmt.Errorf("can't access Slice/Array with a non int Index (%v)", index)
		}
	default:
		err = fmt.Errorf("could not index %T with %T", left, index)
	}

	return err
}

func (c *compiler) evalAccessIndex(left, index interface{}, node *ast.IndexExpression) (interface{}, error) {
	var returnValue interface{}
	var err error
	rv := reflect.ValueOf(left)
	switch rv.Kind() {
	case reflect.Map:
		val := rv.MapIndex(reflect.ValueOf(index))
		if !val.IsValid() {
			return nil, nil
		}

		if node.Callee != nil {
			returnValue, err = c.evalIndexCallee(val, node)
		} else {
			returnValue = val.Interface()
		}
	case reflect.Array, reflect.Slice:
		if i, ok := index.(int); ok {
			if i < 0 || i >= rv.Len() {
				err = fmt.Errorf("array index out of bounds, got index %d, while array size is %d", index, rv.Len())
			} else {
				if node.Callee != nil {
					returnValue, err = c.evalIndexCallee(rv.Index(i), node)
				} else {
					returnValue = rv.Index(i).Interface()
				}
			}
		} else {
			err = fmt.Errorf("can't access Slice/Array with a non int Index (%v)", index)
		}
	default:
		err = fmt.Errorf("could not index %T with %T", left, index)
	}

	return returnValue, err
}

func (c *compiler) evalHashLiteral(node *ast.HashLiteral) (interface{}, error) {
	m := map[string]interface{}{}
	for ke, ve := range node.Pairs {
		v, err := c.evalExpression(ve)
		if err != nil {
			return nil, err
		}

		m[ke.TokenLiteral()] = v
	}

	return m, nil
}

func (c *compiler) evalLetStatement(node *ast.LetStatement) (interface{}, error) {
	v, err := c.evalExpression(node.Value)
	if err != nil {
		return nil, err
	}

	c.ctx.Set(node.Name.Value, v)
	return nil, nil
}

func (c *compiler) evalIdentifier(node *ast.Identifier) (interface{}, error) {
	if node.Callee != nil {
		c, err := c.evalExpression(node.Callee)
		if err != nil {
			return nil, err
		}

		rv := reflect.ValueOf(c)
		if !rv.IsValid() {
			return nil, nil
		}

		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}

		if rv.Kind() != reflect.Struct {
			return nil, fmt.Errorf("'%s' does not have a field or method named '%s' (%s)", node.Callee.String(), node.Value, node)
		}

		f := rv.FieldByName(node.Value)
		if f.Kind() == reflect.Ptr {
			if f.IsNil() {
				return nil, nil
			}

			f = f.Elem()
		}

		if !f.IsValid() {
			m := rv.MethodByName(node.Value)
			if !m.IsValid() {
				return nil, fmt.Errorf("'%s' does not have a field or method named '%s' (%s)", node.Callee.String(), node.Value, node)
			}

			return m.Interface(), nil
		}

		if !f.CanInterface() {
			return nil, fmt.Errorf("'%s'cannot return value obtained from unexported field or method '%s' (%s)", node.Callee.String(), node.Value, node)
		}

		return f.Interface(), nil
	}

	if c.ctx.Has(node.Value) {
		return c.ctx.Value(node.Value), nil
	}

	if node.Value == "nil" {
		return nil, nil
	}

	return nil, &ErrUnknownIdentifier{
		ID: node.Value,
	}
}

func (c *compiler) evalInfixExpression(node *ast.InfixExpression) (interface{}, error) {
	lres, err := c.evalExpression(node.Left)
	if err != nil &&
		node.Operator != "==" && node.Operator != "!=" &&
		node.Operator != "||" && node.Operator != "&&" {
		return nil, err
	} // nil lres is acceptable only for '==', '!=', and logical operators

	switch { // fast return
	case node.Operator == "&&" && !c.isTruthy(lres):
		return false, nil
	case node.Operator == "||" && c.isTruthy(lres):
		return true, nil
	}

	rres, err := c.evalExpression(node.Right)
	if err != nil &&
		node.Operator != "==" && node.Operator != "!=" &&
		node.Operator != "||" && node.Operator != "&&" {
		return nil, err
	} // nil rres is acceptable only for '==', '!=', and logical operators

	switch node.Operator {
	case "&&", "||":
		return c.isTruthy(rres), nil
	} // fast return or this. '&&' and '||' end here

	if nil == lres || nil == rres {
		return c.nilsOperator(lres, rres, node.Operator)
	}

	switch t := lres.(type) {
	case string:
		return c.stringsOperator(t, rres, node.Operator)
	case int64:
		if r, ok := rres.(int64); ok {
			return c.intsOperator(int(t), int(r), node.Operator)
		}
	case int:
		if r, ok := rres.(int); ok {
			return c.intsOperator(t, r, node.Operator)
		}
	case float64:
		if r, ok := rres.(float64); ok {
			return c.floatsOperator(t, r, node.Operator)
		}
	case bool:
		return c.boolsOperator(lres, rres, node.Operator)
	}

	return nil, fmt.Errorf("unable to operate (%s) on %T and %T ", node.Operator, lres, rres)
}

func (c *compiler) nilsOperator(l interface{}, r interface{}, op string) (interface{}, error) {
	switch op {
	case "!=":
		return l != r, nil
	case "==":
		return l == r, nil
	default:
		return nil, fmt.Errorf("unknown operator '%s' on '%T' and '%T' ", op, l, r)
	}
}

func (c *compiler) boolsOperator(l interface{}, r interface{}, op string) (interface{}, error) {
	lt := c.isTruthy(l)
	rt := c.isTruthy(r)

	switch op {
	case "&&", "+":
		return lt && rt, nil
	case "||":
		return lt || rt, nil
	case "!=":
		return lt != rt, nil
	case "==":
		return lt == rt, nil
	default:
		return nil, fmt.Errorf("unknown operator (%s) on %T and %T ", op, lt, rt)
	}
}

func (c *compiler) intsOperator(l int, r int, op string) (interface{}, error) {
	switch op {
	case "+":
		return l + r, nil
	case "-":
		return l - r, nil
	case "/":
		return l / r, nil
	case "*":
		return l * r, nil
	case "<":
		return l < r, nil
	case ">":
		return l > r, nil
	case "!=":
		return l != r, nil
	case ">=":
		return l >= r, nil
	case "<=":
		return l <= r, nil
	case "==":
		return l == r, nil
	}
	return nil, fmt.Errorf("unknown operator for integer %s", op)
}

func (c *compiler) floatsOperator(l float64, r float64, op string) (interface{}, error) {
	switch op {
	case "+":
		return l + r, nil
	case "-":
		return l - r, nil
	case "/":
		return l / r, nil
	case "*":
		return l * r, nil
	case "<":
		return l < r, nil
	case ">":
		return l > r, nil
	case "!=":
		return l != r, nil
	case ">=":
		return l >= r, nil
	case "<=":
		return l <= r, nil
	case "==":
		return l == r, nil
	}
	return nil, fmt.Errorf("unknown operator for float %s", op)
}

func (c *compiler) stringsOperator(l string, r interface{}, op string) (interface{}, error) {
	rr := fmt.Sprint(r)

	switch op {
	case "+":
		return l + rr, nil
	case "<":
		return l < rr, nil
	case ">":
		return l > rr, nil
	case "!=":
		return l != rr, nil
	case ">=":
		return l >= rr, nil
	case "<=":
		return l <= rr, nil
	case "==":
		return l == rr, nil
	case "~=":
		x, err := regexp.Compile(rr)
		if err != nil {
			return nil, fmt.Errorf("couldn't compile regex %s", rr)
		}
		return x.MatchString(l), nil
	}
	return nil, fmt.Errorf("unknown operator for string %s", op)
}

func (c *compiler) evalCallExpression(node *ast.CallExpression) (interface{}, error) {
	var rv reflect.Value

	if node.Callee != nil {
		c, err := c.evalExpression(node.Callee)
		if err != nil {
			return nil, err
		}

		rc := reflect.ValueOf(c)
		mname := node.Function.String()
		if i, ok := node.Function.(*ast.Identifier); ok {
			mname = i.Value
		}

		rv = rc.MethodByName(mname)
		if !rv.IsValid() && rc.Type().Kind() != reflect.Ptr {
			ptr := reflect.New(reflect.TypeOf(c))
			ptr.Elem().Set(rc)
			rv = ptr.MethodByName(mname)
		}

		if !rv.IsValid() {
			if rv.Kind() == reflect.Slice {
				rv = rc.FieldByName(mname)
				if rv.IsValid() {
					return rv.Interface(), nil
				}
			}

			return rc.Interface(), nil
		}
	} else {
		f, err := c.evalExpression(node.Function)
		if err != nil {
			return nil, err
		}

		if ff, ok := f.(*userFunction); ok {
			return c.evalUserFunction(ff, node.Arguments)
		}

		rv = reflect.ValueOf(f)
	}

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if !rv.IsValid() {
		return nil, fmt.Errorf("%+v (%T) is an invalid function", node.String(), rv)
	}

	rt := rv.Type()
	if rt.Kind() != reflect.Func {
		return nil, fmt.Errorf("%+v (%T) is an invalid function", node.String(), rt)
	}

	rtNumIn := rt.NumIn()
	isVariadic := rt.IsVariadic()
	args := []reflect.Value{}

	if !isVariadic {
		if len(node.Arguments) > rtNumIn {
			return nil, fmt.Errorf("%s too many arguments (%d for %d)", node.String(), len(node.Arguments), rtNumIn)
		}

		for pos, a := range node.Arguments {
			v, err := c.evalExpression(a)
			if err != nil {
				return nil, err
			}

			var ar reflect.Value
			expectedT := rt.In(pos)
			if v != nil {
				ar = reflect.ValueOf(v)
			} else {
				ar = reflect.New(expectedT).Elem()
			}

			actualT := ar.Type()
			if !actualT.AssignableTo(expectedT) {
				return nil, fmt.Errorf("%+v (%T) is an invalid argument for %s at pos %d: expected (%s)", v, v, node.Function.String(), pos, expectedT)
			}

			args = append(args, ar)
		}

		hc := func(arg reflect.Type) {
			hhc := reflect.TypeOf((*hctx.HelperContext)(nil)).Elem()
			if arg.ConvertibleTo(reflect.TypeOf(HelperContext{})) || arg.Implements(hhc) {
				hargs := HelperContext{
					Context:  c.ctx,
					compiler: c,
					block:    node.Block,
				}
				args = append(args, reflect.ValueOf(hargs))
				return
			}

			if arg.ConvertibleTo(reflect.TypeOf(map[string]interface{}{})) {
				args = append(args, reflect.ValueOf(map[string]interface{}{}))
				return
			}

			rv := reflect.Indirect(reflect.New(arg))
			args = append(args, rv)
		}

		if len(args) < rtNumIn {
			// missing some args, let's see if we can figure out what they are.
			diff := rtNumIn - len(args)
			switch diff {
			case 2:
				// check last is help
				// check if last -1 is map
				arg := rt.In(rtNumIn - 2)
				hc(arg)
				last := rt.In(rtNumIn - 1)
				hc(last)
			case 1:
				// check if help or map
				last := rt.In(rtNumIn - 1)
				hc(last)
			}
		}

		if len(args) > rtNumIn {
			return nil, fmt.Errorf("%s too many arguments (%d for %d) - %+v", node.String(), len(args), rtNumIn, args)
		}

		if len(args) < rtNumIn {
			return nil, fmt.Errorf("%s too few arguments (%d for %d) - %+v", node.String(), len(args), rtNumIn, args)
		}
	} else {
		// Variadic func
		nodeArgs := node.Arguments
		nodeArgsLen := len(nodeArgs)
		if nodeArgsLen < rtNumIn-1 {
			return nil, fmt.Errorf("%s too few arguments (%d for %d) - %+v", node.String(), len(args), rtNumIn, args)
		}

		var pos int

		// Handle normal args
		for pos = 0; pos < rtNumIn-1; pos++ {
			v, err := c.evalExpression(nodeArgs[pos])
			if err != nil {
				return nil, err
			}

			var ar reflect.Value
			expectedT := rt.In(pos)
			if v != nil {
				ar = reflect.ValueOf(v)
			} else {
				ar = reflect.New(expectedT).Elem()
			}

			actualT := ar.Type()
			if !actualT.AssignableTo(expectedT) {
				return nil, fmt.Errorf("%+v (%T) is an invalid argument for %s at pos %d: expected (%s)", v, v, node.Function.String(), pos, expectedT)
			}

			args = append(args, ar)
		}

		// Unroll variadic arg
		expectedT := rt.In(pos).Elem()
		for ; pos < nodeArgsLen; pos++ {
			v, err := c.evalExpression(nodeArgs[pos])
			if err != nil {
				return nil, err
			}

			var ar reflect.Value
			if v != nil {
				ar = reflect.ValueOf(v)
			} else {
				ar = reflect.New(expectedT)
			}

			actualT := ar.Type()
			if !actualT.AssignableTo(expectedT) {
				return nil, fmt.Errorf("%+v (%T) is an invalid argument for %s at pos %d: expected (%s)", v, v, node.Function.String(), pos, expectedT)
			}

			args = append(args, ar)
		}
	}

	res := rv.Call(args)
	if len(res) > 0 {
		if e, ok := res[len(res)-1].Interface().(error); ok {
			return nil, fmt.Errorf("could not call %s function: %w", node.Function, e)
		}
		return res[0].Interface(), nil
	}

	return nil, nil
}

func (c *compiler) evalForExpression(node *ast.ForExpression) (interface{}, error) {
	octx := c.ctx.(*Context)
	defer func() {
		c.ctx = octx
	}()

	c.ctx = octx.New()
	// must copy all data from original (it includes application defined helpers)
	for k, v := range octx.data {
		c.ctx.Set(k, v)
	}

	iter, err := c.evalExpression(node.Iterable)
	if err != nil {
		return nil, err
	}

	riter := reflect.ValueOf(iter)
	if riter.Kind() == reflect.Ptr {
		riter = riter.Elem()
	}

	ret := []interface{}{}
	switch riter.Kind() {
	case reflect.Map:
		keys := riter.MapKeys()
		for i := 0; i < len(keys); i++ {
			k := keys[i]
			v := riter.MapIndex(k)
			c.ctx.Set(node.KeyName, k.Interface())
			c.ctx.Set(node.ValueName, v.Interface())

			res, err := c.evalBlockStatement(node.Block)
			if err != nil {
				return nil, err
			}

			breakLoop := false
			switch val := res.(type) {
			case continueObject:
				res = val.Value
			case breakObject:
				breakLoop = true
				res = val.Value
			}

			if res != nil {
				ret = append(ret, res)
			}

			if breakLoop {
				break
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < riter.Len(); i++ {
			v := riter.Index(i)
			c.ctx.Set(node.KeyName, i)
			c.ctx.Set(node.ValueName, v.Interface())

			res, err := c.evalBlockStatement(node.Block)
			if err != nil {
				return nil, err
			}

			breakLoop := false
			switch val := res.(type) {
			case continueObject:
				res = val.Value
			case breakObject:
				breakLoop = true
				res = val.Value
			}

			if res != nil {
				ret = append(ret, res)
			}

			if breakLoop {
				break
			}
		}
	default:
		if iter == nil {
			return nil, nil
		}
		if it, ok := iter.(Iterator); ok {
			i := 0
			ii := it.Next()
			for ii != nil {
				c.ctx.Set(node.KeyName, i)
				c.ctx.Set(node.ValueName, ii)

				res, err := c.evalBlockStatement(node.Block)
				if err != nil {
					return nil, err
				}

				breakLoop := false
				switch val := res.(type) {
				case continueObject:
					res = val.Value
				case breakObject:
					breakLoop = true
					res = val.Value
				}

				if res != nil {
					ret = append(ret, res)
				}

				if breakLoop {
					break
				}

				ii = it.Next()
				i++
			}
			return ret, nil
		}
		return ret, fmt.Errorf("could not iterate over %T", iter)
	}
	return ret, nil
}

func (c *compiler) evalBlockStatement(node *ast.BlockStatement) (interface{}, error) {
	res := []interface{}{}
	for _, s := range node.Statements {
		i, err := c.evalStatement(s)
		if err != nil {
			return nil, err
		}

		val, exitBlock := i.(exitBlockStatment)
		if !exitBlock {
			if i != nil {
				res = append(res, i)
			}
		} else {
			var resValue interface{}
			switch obj := val.(type) {
			case continueObject:
				obj = continueObject{Value: append(res, obj.Value...)}
				resValue = obj
			case breakObject:
				obj = breakObject{Value: append(res, obj.Value...)}
				resValue = obj
			case returnObject:
				res = append(res, i)
				obj.Value = res
				resValue = obj
			}

			return resValue, nil
		}
	}

	return res, nil
}

func (c *compiler) evalStatement(node ast.Statement) (interface{}, error) {
	c.curStmt = node

	switch t := node.(type) {
	case *ast.ExpressionStatement:
		s, err := c.evalExpression(t.Expression)
		switch s.(type) {
		case exitBlockStatment, ast.Printable, template.HTML:
			return s, err
		}

		return nil, err
	case *ast.ReturnStatement:
		return c.evalReturnStatement(t)
	case *ast.LetStatement:
		return c.evalLetStatement(t)
	}

	return nil, fmt.Errorf("could not eval statement %T", node)
}

func (c *compiler) evalReturnStatement(node *ast.ReturnStatement) (interface{}, error) {
	res, err := c.evalExpression(node.ReturnValue)
	if err != nil {
		return nil, err
	}

	if node.Type == token.RETURN {
		v := returnObject{}
		v.Value = append(v.Value, res)
		res = v
	}

	return res, nil
}

func (c *compiler) evalArrayLiteral(node *ast.ArrayLiteral) (interface{}, error) {
	res := []interface{}{}

	for _, e := range node.Elements {
		i, err := c.evalExpression(e)
		if err != nil {
			return nil, err
		}

		res = append(res, i)
	}

	return res, nil
}

func (c *compiler) evalIndexCallee(rv reflect.Value, node *ast.IndexExpression) (interface{}, error) {
	octx := c.ctx.(*Context)
	defer func() {
		c.ctx = octx
	}()

	c.ctx = octx.New()
	// must copy all data from original (it includes application defined helpers)
	for k, v := range octx.data {
		c.ctx.Set(k, v)
	}

	//The key here is needed to set the object in ctx for later evaluation
	//For example, if this is a nested object person.Name[0]
	//then we can set the value of Name[0] to person.Name
	//As the evalIdent will look for that object by person.Name
	//If key doesn't contain "." this means we got person[0].Name[0]
	//If key does contain "." then indexed field that needs to be accessed will be set in Node.left and Node.Callee
	key := node.Left.String()
	if strings.Contains(key, ".") {
		ggg := strings.Split(key, ".")
		callee := node.Callee.String()

		if !strings.Contains(callee, key) {
			for {
				if len(ggg) >= 2 {
					ggg = ggg[1:]
				} else {
					key = ggg[0]
					break
				}

				if strings.Contains(callee, strings.Join(ggg, ".")) {
					key = strings.Join(ggg, ".")
					break
				}
			}
		}
	}

	c.ctx.Set(key, rv.Interface())

	vvs, err := c.evalExpression(node.Callee)
	if err != nil {
		return nil, err
	}

	return vvs, nil
}
