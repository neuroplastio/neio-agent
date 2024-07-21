package flowapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi/flowdsl"
)

type Arguments struct {
	parameters []flowdsl.Parameter
	arguments  []flowdsl.Argument
	nameMap    map[string]int
}

func NewArguments(parameters []flowdsl.Parameter, arguments []flowdsl.Argument) (Arguments, error) {
	nameMap := make(map[string]int)
	for i, p := range parameters {
		nameMap[p.Name] = i
	}
	requiredParams := 0
	for _, p := range parameters {
		if p.Default == nil {
			requiredParams++
		}
	}
	if len(arguments) < requiredParams {
		return Arguments{}, fmt.Errorf("not enough arguments provided: %d out of %d", len(arguments), requiredParams)
	}

	if len(arguments) > len(parameters) {
		return Arguments{}, fmt.Errorf("too many arguments provided: %d out of %d", len(arguments), len(parameters))
	}

	for i, param := range parameters {
		if param.Default != nil {
			continue
		}
		arg := arguments[i]
		switch param.Type {
		case "string":
			if arg.Value.String == nil {
				return Arguments{}, fmt.Errorf("argument %d should be a string", i)
			}
		case "number":
			if arg.Value.Number == nil {
				return Arguments{}, fmt.Errorf("argument %d should be a number", i)
			}
		case "boolean":
			if arg.Value.Boolean == nil {
				return Arguments{}, fmt.Errorf("argument %d should be a boolean", i)
			}
		case "Duration":
			if arg.Duration == nil {
				return Arguments{}, fmt.Errorf("argument %d should be a duration", i)
			}
		case "Action":
			if arg.Expr == nil && arg.Usage == nil {
				return Arguments{}, fmt.Errorf("argument %d should be an action", i)
			}
		case "Signal":
			if arg.Expr == nil {
				return Arguments{}, fmt.Errorf("argument %d should be a signal", i)
			}
		case "any":
		default:
			return Arguments{}, fmt.Errorf("unsupported type %s for a parameter: %s", param.Type, param.Name)
		}
	}
	return Arguments{
		parameters: parameters,
		arguments:  arguments,
		nameMap:    nameMap,
	}, nil
}

func (a Arguments) Argument(name string) flowdsl.Argument {
	arg := a.ArgumentOrNil(name)
	if arg == nil {
		return flowdsl.Argument{
			Value: &flowdsl.Value{},
		}
	}
	return *arg
}

func (a Arguments) ArgumentOrNil(name string) *flowdsl.Argument {
	idx, ok := a.nameMap[name]
	if !ok {
		return nil
	}
	if len(a.arguments) <= idx {
		defaultValue := a.parameters[idx].Default
		if defaultValue == nil {
			return nil
		}
		return &flowdsl.Argument{
			Value:    defaultValue.Value,
			Duration: defaultValue.Duration,
		}
	}
	return &a.arguments[idx]
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

func (a Arguments) Usages(name string) []string {
	arg := a.Argument(name)
	if arg.Usage == nil {
		return nil
	}
	return arg.Usage.Usages
}

func (a Arguments) Statement(name string) flowdsl.Statement {
	arg := a.Argument(name)
	return flowdsl.Statement{
		Expr:  arg.Expr,
		Usage: arg.Usage,
	}
}

func (a Arguments) StatementOrNil(name string) *flowdsl.Statement {
	arg := a.ArgumentOrNil(name)
	if arg == nil {
		return nil
	}
	if arg.Expr == nil && arg.Usage == nil {
		return nil
	}
	return &flowdsl.Statement{
		Expr:  arg.Expr,
		Usage: arg.Usage,
	}
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
