package flowdsl

import (
	"encoding/json"
	"fmt"
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
			case "Action", "Signal", "Usage":
				if p.Default.Value == nil || !p.Default.Value.IsNull() {
					return Declaration{}, fmt.Errorf("parameter %s default value can only be 'null'", p.Name)
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
