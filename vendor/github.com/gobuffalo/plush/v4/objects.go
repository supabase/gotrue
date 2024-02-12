package plush

type exitBlockStatment interface {
	exitBlock()
}

type returnObject struct {
	exitBlockStatment
	Value []interface{}
}

type continueObject struct {
	exitBlockStatment
	Value []interface{}
}

type breakObject struct {
	exitBlockStatment
	Value []interface{}
}
