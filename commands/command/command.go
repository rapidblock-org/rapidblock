package command

import (
	getopt "github.com/pborman/getopt/v2"
)

type Command struct {
	Name        string
	Aliases     []string
	Description string
	Options     *getopt.Set
	Main        func() int
}

type Factory interface {
	New() Command
}

type FactoryFunc func() Command

func (fn FactoryFunc) New() Command {
	return fn()
}

var _ Factory = FactoryFunc(nil)
