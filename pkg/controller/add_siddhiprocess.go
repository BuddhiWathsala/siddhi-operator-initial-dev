package controller

import (
	"github.com/siddhi-io/siddhi-operator/pkg/controller/siddhiprocess"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, siddhiprocess.Add)
}
