package actiondsl

import (
	"encoding/json"
	"fmt"
	"time"
)

func ParseStatement(stmt string) (Statement, error) {
	result, err := statementParser.ParseString("", stmt)
	if err != nil {
		return Statement{}, err
	}
	return *result, nil
}

func ParseDeclaration(decl string) (Declaration, error) {
	result, err := declarationParser.ParseString("", decl)
	if err != nil {
		return Declaration{}, err
	}
	shouldHaveDefault := false
	for _, p := range result.Parameters {
		if shouldHaveDefault && p.Default == nil {
			return Declaration{}, fmt.Errorf("parameter %s should have a default value", p.Name)
		}
		if p.Default != nil {
			shouldHaveDefault = true
			switch p.Type {
			case "string":
				if p.Default.Value.String == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a string", p.Name)
				}
			case "number":
				if p.Default.Value.Number == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a number", p.Name)
				}
			case "boolean":
				if p.Default.Value.Boolean == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a boolean", p.Name)
				}
			case "Duration":
				if p.Default.Duration == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a duration", p.Name)
				}
			case "any":
			default:
				return Declaration{}, fmt.Errorf("unsupported type %s for a default value: %s", p.Type, p.Name)
			}
		}
	}
	return *result, nil
}

func ParseUsages(expr string) ([]string, error) {
	result, err := usageParser.ParseString("", expr)
	if err != nil {
		return nil, err
	}
	return result.Usages, nil
}

type Action struct {
	Declaration

	statement Statement
	arguments Arguments
}

func NewAction(decl Declaration, stmt Statement) (Action, error) {
	if decl.Action != stmt.Action {
		return Action{}, fmt.Errorf("declaration and statement actions do not match")
	}

	requiredParams := 0
	for _, p := range decl.Parameters {
		if p.Default == nil {
			requiredParams++
		}
	}
	if len(stmt.Arguments) < requiredParams {
		return Action{}, fmt.Errorf("not enough arguments provided: %d out of %d", len(stmt.Arguments), requiredParams)
	}

	for i, param := range decl.Parameters {
		if param.Default != nil {
			continue
		}
		arg := stmt.Arguments[i]
		switch param.Type {
		case "string":
			if arg.Value.String == nil {
				return Action{}, fmt.Errorf("argument %d should be a string", i)
			}
		case "number":
			if arg.Value.Number == nil {
				return Action{}, fmt.Errorf("argument %d should be a number", i)
			}
		case "boolean":
			if arg.Value.Boolean == nil {
				return Action{}, fmt.Errorf("argument %d should be a boolean", i)
			}
		case "Duration":
			if arg.Duration == nil {
				return Action{}, fmt.Errorf("argument %d should be a duration", i)
			}
		case "Action":
			if arg.Action == nil && arg.Action.Usages == nil {
				return Action{}, fmt.Errorf("argument %d should be an action", i)
			}
		case "any":
		default:
			return Action{}, fmt.Errorf("unsupported type %s for a parameter: %s", param.Type, param.Name)
		}
	}

	return Action{
		Declaration: decl,
		statement:   stmt,
		arguments:   NewArguments(decl.Parameters, stmt.Arguments),
	}, nil
}

func (a Action) Args() Arguments {
	return a.arguments
}

type Arguments struct {
	parameters []Parameter
	arguments  []Argument
	nameMap    map[string]int
}

func NewArguments(parameters []Parameter, arguments []Argument) Arguments {
	nameMap := make(map[string]int)
	for i, p := range parameters {
		nameMap[p.Name] = i
	}
	return Arguments{
		parameters: parameters,
		arguments:  arguments,
		nameMap:    nameMap,
	}
}

func (a Arguments) Argument(name string) Argument {
	idx, ok := a.nameMap[name]
	if !ok {
		return Argument{
			Value: &Value{},
		}
	}
	if len(a.arguments) <= idx {
		defaultValue := a.parameters[idx].Default
		if defaultValue == nil {
			return Argument{
				Value: &Value{},
			}
		}
		return Argument{
			Value:    defaultValue.Value,
			Duration: defaultValue.Duration,
		}
	}
	return a.arguments[idx]
}

func (a Arguments) String(name string) string {
	arg := a.Argument(name)
	if arg.Value == nil || arg.Value.String == nil {
		return ""
	}
	return *arg.Value.String
}

func (a Arguments) Number(name string) json.Number {
	arg := a.Argument(name)
	if arg.Value == nil || arg.Value.Number == nil {
		return json.Number("0")
	}
	return json.Number(*arg.Value.Number)
}

func (a Arguments) Duration(name string) time.Duration {
	arg := a.Argument(name)
	if arg.Duration == nil {
		return 0
	}
	return time.Duration(*arg.Duration)
}

func (a Arguments) Action(name string) Statement {
	arg := a.Argument(name)
	if arg.Action == nil {
		return Statement{}
	}
	return *arg.Action
}

func (a Arguments) Int(name string) int {
	num := a.Number(name)
	i64, _ := num.Int64()
	return int(i64)
}

func (a Arguments) Float(name string) float64 {
	num := a.Number(name)
	f64, _ := num.Float64()
	return f64
}

func (a Arguments) Boolean(name string) bool {
	arg := a.Argument(name)
	if arg.Value == nil || arg.Value.Boolean == nil {
		return false
	}
	return bool(*arg.Value.Boolean)
}

func (a Arguments) Any(name string) any {
	arg := a.Argument(name)
	if arg.Duration != nil {
		return time.Duration(*arg.Duration)
	}
	if arg.Value != nil {
		switch {
		case arg.Value.String != nil:
			return *arg.Value.String
		case arg.Value.Number != nil:
			f64, err := json.Number(*arg.Value.Number).Float64()
			if err != nil {
				return nil
			}
			return f64
		case arg.Value.Boolean != nil:
			return bool(*arg.Value.Boolean)
		}
	}
	return nil
}
