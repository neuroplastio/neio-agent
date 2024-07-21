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
				if p.Default.Value == nil || p.Default.Value.String == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a string", p.Name)
				}
			case "number":
				if p.Default.Value == nil || p.Default.Value.Number == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a number", p.Name)
				}
			case "boolean":
				if p.Default.Value == nil || p.Default.Value.Boolean == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a boolean", p.Name)
				}
			case "Duration":
				if p.Default.Duration == nil {
					return Declaration{}, fmt.Errorf("parameter %s default value should be a duration", p.Name)
				}
			case "Action", "Signal":
				if p.Default.Value == nil || !p.Default.Value.IsNull() {
					return Declaration{}, fmt.Errorf("default parameter for Action and Signal types can only be 'null'", p.Name)
				}
			case "any":
			default:
				return Declaration{}, fmt.Errorf("unsupported type %s for a default value: %s", p.Type, p.Name)
			}
		}
	}
	return *result, nil
}

func ParseUsageStatement(expr string) (UsageStatement, error) {
	result, err := usageParser.ParseString("", expr)
	if err != nil {
		return UsageStatement{}, err
	}
	return *result, nil
}

type DeclarationCall struct {
	declaration Declaration
	statement   ExpressionStatement
	arguments   Arguments
}

func NewDeclarationCall(decl Declaration, stmt ExpressionStatement) (DeclarationCall, error) {
	// TODO: replace Call with just args
	requiredParams := 0
	for _, p := range decl.Parameters {
		if p.Default == nil {
			requiredParams++
		}
	}
	if len(stmt.Arguments) < requiredParams {
		return DeclarationCall{}, fmt.Errorf("not enough arguments provided: %d out of %d", len(stmt.Arguments), requiredParams)
	}

	for i, param := range decl.Parameters {
		if param.Default != nil {
			continue
		}
		arg := stmt.Arguments[i]
		switch param.Type {
		case "string":
			if arg.Value.String == nil {
				return DeclarationCall{}, fmt.Errorf("argument %d should be a string", i)
			}
		case "number":
			if arg.Value.Number == nil {
				return DeclarationCall{}, fmt.Errorf("argument %d should be a number", i)
			}
		case "boolean":
			if arg.Value.Boolean == nil {
				return DeclarationCall{}, fmt.Errorf("argument %d should be a boolean", i)
			}
		case "Duration":
			if arg.Duration == nil {
				return DeclarationCall{}, fmt.Errorf("argument %d should be a duration", i)
			}
		case "Action":
			if arg.Expr == nil && arg.Usage == nil {
				return DeclarationCall{}, fmt.Errorf("argument %d should be an action", i)
			}
		case "Signal":
			if arg.Expr == nil {
				return DeclarationCall{}, fmt.Errorf("argument %d should be a signal", i)
			}
		case "any":
		default:
			return DeclarationCall{}, fmt.Errorf("unsupported type %s for a parameter: %s", param.Type, param.Name)
		}
	}

	return DeclarationCall{
		declaration: decl,
		statement:   stmt,
		arguments:   NewArguments(decl.Parameters, stmt.Arguments),
	}, nil
}

func (a DeclarationCall) Args() Arguments {
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
	arg := a.ArgumentOrNil(name)
	if arg == nil {
		return Argument{
			Value: &Value{},
		}
	}
	return *arg
}

func (a Arguments) ArgumentOrNil(name string) *Argument {
	idx, ok := a.nameMap[name]
	if !ok {
		return nil
	}
	if len(a.arguments) <= idx {
		defaultValue := a.parameters[idx].Default
		if defaultValue == nil {
			return nil
		}
		return &Argument{
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

func (a Arguments) Expression(name string) ExpressionStatement {
	arg := a.Argument(name)
	if arg.Expr == nil {
		return ExpressionStatement{}
	}
	return *arg.Expr
}

func (a Arguments) ExpressionOrNil(name string) *ExpressionStatement {
	arg := a.ArgumentOrNil(name)
	if arg == nil {
		return nil
	}
	if arg.Expr == nil {
		return nil
	}
	return arg.Expr
}

func (a Arguments) Statement(name string) Statement {
	arg := a.Argument(name)
	return Statement{
		Expr:  arg.Expr,
		Usage: arg.Usage,
	}
}

func (a Arguments) StatementOrNil(name string) *Statement {
	arg := a.ArgumentOrNil(name)
	if arg == nil {
		return nil
	}
	if arg.Expr == nil && arg.Usage == nil {
		return nil
	}
	return &Statement{
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

type JSONExpressionMapItem struct {
	Usage     UsageStatement
	Statement Statement

	UsageString     string
	StatementString string
}

type JSONExpressionItems []JSONExpressionMapItem

func (j *JSONExpressionItems) UnmarshalJSON(data []byte) error {
	strings := make(map[string]string)
	if err := json.Unmarshal(data, &strings); err != nil {
		return err
	}
	items := make([]JSONExpressionMapItem, 0, len(strings))
	for k, v := range strings {
		usage, err := ParseUsageStatement(k)
		if err != nil {
			return err
		}
		stmt, err := ParseStatement(v)
		if err != nil {
			return err
		}
		items = append(items, JSONExpressionMapItem{
			Usage:           usage,
			Statement:       stmt,
			UsageString:     k,
			StatementString: v,
		})
	}
	*j = JSONExpressionItems(items)
	return nil
}
