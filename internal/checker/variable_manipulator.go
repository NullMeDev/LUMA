package checker

import (
	"fmt"
	"strconv"
	"strings"
	"regexp"
)

// VariableManipulator provides advanced variable manipulation capabilities
type VariableManipulator struct {
	variables *VariableList
}

// NewVariableManipulator creates a new variable manipulator
func NewVariableManipulator(variables *VariableList) *VariableManipulator {
	return &VariableManipulator{
		variables: variables,
	}
}

// ReplaceVariables replaces variable placeholders in text with actual values
func (vm *VariableManipulator) ReplaceVariables(text string) string {
	// Handle standard variables <VAR>, <VAR[index]>, <VAR(key)>
	re := regexp.MustCompile(`<([^<>]+)>`)
	
	return re.ReplaceAllStringFunc(text, func(match string) string {
		// Remove < and >
		varExpr := match[1 : len(match)-1]
		
		// Check for array index: VAR[0]
		if strings.Contains(varExpr, "[") && strings.Contains(varExpr, "]") {
			return vm.handleArrayVariable(varExpr)
		}
		
		// Check for dictionary key: VAR(key)
		if strings.Contains(varExpr, "(") && strings.Contains(varExpr, ")") {
			return vm.handleDictionaryVariable(varExpr)
		}
		
		// Simple variable
		if variable, err := vm.variables.Get(varExpr); err == nil {
			return vm.variableToString(variable)
		}
		
		return match // Return original if variable not found
	})
}

// handleArrayVariable processes array variable references like VAR[0]
func (vm *VariableManipulator) handleArrayVariable(varExpr string) string {
	parts := strings.Split(varExpr, "[")
	if len(parts) != 2 {
		return "<" + varExpr + ">"
	}
	
	varName := parts[0]
	indexPart := strings.TrimSuffix(parts[1], "]")
	
	variable, err := vm.variables.Get(varName)
	if err != nil || variable.VarType != List {
		return "<" + varExpr + ">"
	}
	
	index, err := strconv.Atoi(indexPart)
	if err != nil {
		return "<" + varExpr + ">"
	}
	
	list, ok := variable.Value.([]string)
	if !ok {
		return "<" + varExpr + ">"
	}
	
	if index < 0 || index >= len(list) {
		return "<" + varExpr + ">"
	}
	
	return list[index]
}

// handleDictionaryVariable processes dictionary variable references like VAR(key)
func (vm *VariableManipulator) handleDictionaryVariable(varExpr string) string {
	parts := strings.Split(varExpr, "(")
	if len(parts) != 2 {
		return "<" + varExpr + ">"
	}
	
	varName := parts[0]
	keyPart := strings.TrimSuffix(parts[1], ")")
	
	variable, err := vm.variables.Get(varName)
	if err != nil || variable.VarType != Dictionary {
		return "<" + varExpr + ">"
	}
	
	dict, ok := variable.Value.(map[string]string)
	if !ok {
		return "<" + varExpr + ">"
	}
	
	if value, exists := dict[keyPart]; exists {
		return value
	}
	
	return "<" + varExpr + ">"
}

// variableToString converts a variable to its string representation
func (vm *VariableManipulator) variableToString(variable *Variable) string {
	switch variable.VarType {
	case Single:
		return fmt.Sprintf("%v", variable.Value)
	case List:
		if list, ok := variable.Value.([]string); ok {
			return strings.Join(list, ",")
		}
		return fmt.Sprintf("%v", variable.Value)
	case Dictionary:
		if dict, ok := variable.Value.(map[string]string); ok {
			var pairs []string
			for k, v := range dict {
				pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
			}
			return strings.Join(pairs, ",")
		}
		return fmt.Sprintf("%v", variable.Value)
	default:
		return fmt.Sprintf("%v", variable.Value)
	}
}

// SetVariable sets a variable with type inference
func (vm *VariableManipulator) SetVariable(name string, value interface{}, isCapture bool) {
	var varType VariableType
	
	switch v := value.(type) {
	case []string:
		varType = List
	case map[string]string:
		varType = Dictionary
	case string:
		varType = Single
	default:
		varType = Single
		value = fmt.Sprintf("%v", v)
	}
	
	variable := &Variable{
		Name:      name,
		Value:     value,
		VarType:   varType,
		IsCapture: isCapture,
	}
	
	vm.variables.Set(variable)
}

// TransformVariable applies a transformation to a variable
func (vm *VariableManipulator) TransformVariable(varName string, funcType FunctionType, params ...string) error {
	variable, err := vm.variables.Get(varName)
	if err != nil {
		return err
	}
	
	// Only transform single variables for now
	if variable.VarType != Single {
		return fmt.Errorf("can only transform single variables")
	}
	
	input := fmt.Sprintf("%v", variable.Value)
	functionBlock := &FunctionBlock{}
	
	result, err := functionBlock.Apply(funcType, input, params...)
	if err != nil {
		return err
	}
	
	// Update the variable with the transformed value
	variable.Value = result
	vm.variables.Set(variable)
	
	return nil
}

// GetCapturedVariables returns all variables marked as captures
func (vm *VariableManipulator) GetCapturedVariables() map[string]*Variable {
	captured := make(map[string]*Variable)
	
	for _, varName := range vm.variables.List() {
		if variable, err := vm.variables.Get(varName); err == nil && variable.IsCapture {
			captured[varName] = variable
		}
	}
	
	return captured
}
