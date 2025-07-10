package checker

import (
	"fmt"
)

// WorkflowStep represents a single step in a parsing workflow
type WorkflowStep struct {
	Type       string   `json:"type"`        // "parse" or "function"
	ParseType  string   `json:"parse_type"`  // for parsing steps
	FuncType   string   `json:"func_type"`   // for function steps
	Parameters []string `json:"parameters"`  // parameters for the operation
	OutputVar  string   `json:"output_var"`  // variable name to store result
}

// Workflow represents a sequence of parsing and transformation steps
type Workflow struct {
	Name  string         `json:"name"`
	Steps []WorkflowStep `json:"steps"`
}

// WorkflowEngine manages and executes workflows
type WorkflowEngine struct {
	parsingEngine *ParsingEngine
	functionBlock *FunctionBlock
	variables     *VariableList
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine() *WorkflowEngine {
	return &WorkflowEngine{
		parsingEngine: NewParsingEngine(),
		functionBlock: &FunctionBlock{},
		variables:     NewVariableList(),
	}
}

// Execute runs a workflow on the given input
func (we *WorkflowEngine) Execute(workflow Workflow, input string) error {
	// Set initial input as SOURCE variable
	we.variables.Set(&Variable{
		Name:    "SOURCE",
		Value:   input,
		VarType: Single,
	})

	for stepIndex, step := range workflow.Steps {
		var result []string
		var err error

		// Get input for this step
		stepInput := input
		if stepIndex > 0 {
			// Use previous step's output or SOURCE
			if sourceVar, err := we.variables.Get("SOURCE"); err == nil {
				if sourceVar.VarType == Single {
					stepInput = sourceVar.Value.(string)
				}
			}
		}

		switch step.Type {
		case "parse":
			parseType := ParseType(step.ParseType)
			result, err = we.parsingEngine.Parse(parseType, stepInput, step.Parameters...)
			if err != nil {
				return fmt.Errorf("parsing step %d failed: %v", stepIndex, err)
			}

		case "function":
			funcType := FunctionType(step.FuncType)
			singleResult, err := we.functionBlock.Apply(funcType, stepInput, step.Parameters...)
			if err != nil {
				return fmt.Errorf("function step %d failed: %v", stepIndex, err)
			}
			result = []string{singleResult}

		default:
			return fmt.Errorf("unknown step type: %s", step.Type)
		}

		// Store result in variable
		var variable *Variable
		if len(result) == 1 {
			variable = &Variable{
				Name:      step.OutputVar,
				Value:     result[0],
				VarType:   Single,
				IsCapture: true,
			}
		} else {
			variable = &Variable{
				Name:      step.OutputVar,
				Value:     result,
				VarType:   List,
				IsCapture: true,
			}
		}

		we.variables.Set(variable)

		// Update SOURCE for next step
		if len(result) > 0 {
			we.variables.Set(&Variable{
				Name:    "SOURCE",
				Value:   result[0],
				VarType: Single,
			})
		}
	}

	return nil
}

// GetVariable retrieves a variable from the workflow execution
func (we *WorkflowEngine) GetVariable(name string) (*Variable, error) {
	return we.variables.Get(name)
}

// GetAllVariables returns all variables from the workflow execution
func (we *WorkflowEngine) GetAllVariables() map[string]*Variable {
	allVars := make(map[string]*Variable)
	for _, varName := range we.variables.List() {
		if variable, err := we.variables.Get(varName); err == nil {
			allVars[varName] = variable
		}
	}
	return allVars
}

// Reset clears all variables for a fresh workflow execution
func (we *WorkflowEngine) Reset() {
	we.variables = NewVariableList()
}
