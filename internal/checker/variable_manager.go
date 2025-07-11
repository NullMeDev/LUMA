package checker

import (
	"fmt"
	"sync"
)

// VariableType defines the type of variable being stored
type VariableType int

const (
	Single VariableType = iota
	List
	Dictionary
)

// Variable holds a generic variable in the checker
type Variable struct {
	Name      string
	Value     interface{}
	VarType   VariableType
	IsCapture bool
}

// VariableList manages a list of variables
type VariableList struct {
	variables map[string]*Variable
	mu        sync.RWMutex
}

// NewVariableList initializes a VariableList
func NewVariableList() *VariableList {
	return &VariableList{
		variables: make(map[string]*Variable),
	}
}

// Set sets a variable in the list
func (vl *VariableList) Set(variable *Variable) {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	vl.variables[variable.Name] = variable
}

// Get retrieves a variable by name
func (vl *VariableList) Get(name string) (*Variable, error) {
	vl.mu.RLock()
	defer vl.mu.RUnlock()
	variable, exists := vl.variables[name]
	if !exists {
		return nil, fmt.Errorf("variable %s not found", name)
	}
	return variable, nil
}

// Remove deletes a variable by name
func (vl *VariableList) Remove(name string) {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	delete(vl.variables, name)
}

// List returns all variable names
func (vl *VariableList) List() []string {
	vl.mu.RLock()
	defer vl.mu.RUnlock()
	keys := make([]string, 0, len(vl.variables))
	for k := range vl.variables {
		keys = append(keys, k)
	}
	return keys
}
