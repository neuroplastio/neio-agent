package flowdsl

import (
	"encoding/json"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	ruleIdent          = lexer.SimpleRule{Name: "Ident", Pattern: `[a-z][\w\d]*`}
	ruleUsageIdent     = lexer.SimpleRule{Name: "UsageIdent", Pattern: `((key|btn)\.)?[0-9A-Z][\w\d]*`}
	ruleUsageKey       = lexer.SimpleRule{Name: "UsageKey", Pattern: `[0-9]|([A-Z]\w*)`}
	ruleType           = lexer.SimpleRule{Name: "Type", Pattern: `(string|number|boolean|any|Duration|Action|Signal)`}
	ruleDuration       = lexer.SimpleRule{Name: "Duration", Pattern: `\d+(ns|us|µs|ms|s|m|h)`}
	ruleString         = lexer.SimpleRule{Name: "String", Pattern: `"(\\"|[^"])*"`}
	ruleNumber         = lexer.SimpleRule{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`}
	rulePunct          = lexer.SimpleRule{Name: "Punct", Pattern: `[-[!@#$%^&*()+_={}\|:;"'<,>.?/]|]`}
	ruleWhitespace     = lexer.SimpleRule{Name: "Whitespace", Pattern: `[ \t]+`}
	ruleReferenceIdent = lexer.SimpleRule{Name: "ReferenceIdent", Pattern: `\$[a-z][\w\d]*\.[a-z][\w\d]*`}
)

var statementLexer = lexer.MustSimple([]lexer.SimpleRule{
	ruleWhitespace,
	ruleDuration,
	ruleString,
	ruleNumber,
	ruleUsageIdent,
	ruleReferenceIdent,
	ruleIdent,
	rulePunct,
})

var statementParser = participle.MustBuild[Statement](
	participle.Lexer(statementLexer),
	participle.UseLookahead(2),
	participle.Elide(ruleWhitespace.Name),
	participle.Unquote("String"),
)

var usageParser = participle.MustBuild[UsageStatement](
	participle.Lexer(statementLexer),
	participle.UseLookahead(2),
	participle.Elide(ruleWhitespace.Name),
	participle.Unquote("String"),
)

type Statement struct {
	Usage *UsageStatement      `parser:"@@ |" json:"usage,omitempty"`
	Expr  *ExpressionStatement `parser:"@@" json:"expr,omitempty"`
}

type UsageStatement struct {
	Usages []string `parser:"@UsageIdent ('+' @UsageIdent)*" json:"usages,omitempty"`
}

type ExpressionStatement struct {
	Identifier string     `parser:"(@ReferenceIdent | @Ident)" json:"identifier"`
	Arguments  []Argument `parser:"'(' @@? (',' @@)* ')'" json:"arguments,omitempty"`
}

type Argument struct {
	Usage    *UsageStatement      `parser:"@@ |" json:"usage,omitempty"`
	Expr     *ExpressionStatement `parser:"@@ |" json:"expr,omitempty"`
	Duration *Duration            `parser:"@Duration |" json:"duration,omitempty"`
	Value    *Value               `parser:"@@" json:"value,omitempty"`
}

type Value struct {
	String  *string  `parser:"@String |"`
	Number  *Number  `parser:"@Number |"`
	Boolean *Boolean `parser:"@('true'|'false') |"`
	Null    *Null    `parser:"@('null')"`
}

type Null struct{}

func (n *Null) Capture(values []string) error {
	*n = Null{}
	return nil
}

func (v *Value) IsNull() bool {
	return v.String == nil && v.Number == nil && v.Boolean == nil
}

func (v *Value) MarshalJSON() ([]byte, error) {
	if v.String != nil {
		return json.Marshal(v.String)
	}
	if v.Number != nil {
		return json.Marshal(json.Number(*v.Number))
	}
	if v.Boolean != nil {
		return json.Marshal(bool(*v.Boolean))
	}
	return []byte("null"), nil
}

func (v *Value) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "null":
		*v = Value{}
		return nil
	case "true":
		b := Boolean(true)
		*v = Value{Boolean: &b}
		return nil
	case "false":
		b := Boolean(false)
		*v = Value{Boolean: &b}
	}
	if len(data) > 2 && data[0] == '"' && data[len(data)-1] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*v = Value{String: &str}
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*v = Value{Number: (*Number)(&num)}
	return nil
}

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type Number json.Number

func (n *Number) Capture(values []string) error {
	*n = Number(values[0])
	return nil
}

type Duration time.Duration

func (d *Duration) Capture(values []string) error {
	duration, err := time.ParseDuration(values[0])
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

var declarationLexer = lexer.MustSimple([]lexer.SimpleRule{
	ruleWhitespace,
	ruleType,
	ruleIdent,
	ruleString,
	ruleDuration,
	ruleNumber,
	rulePunct,
})

var declarationParser = participle.MustBuild[Declaration](
	participle.Lexer(declarationLexer),
	participle.UseLookahead(2),
	participle.Elide(ruleWhitespace.Name),
	participle.Unquote("String"),
)

type Declaration struct {
	Identifier string      `parser:"@Ident" json:"action"`
	Parameters []Parameter `parser:"'(' ( @@ ( ',' @@ )* )? ')'" json:"parameters,omitempty"`
}

type Parameter struct {
	Name    string          `parser:"@Ident" json:"name"`
	Type    string          `parser:"':' @Type" json:"type"`
	Default *ParameterValue `parser:"('=' @@)?" json:"default,omitempty"`
}

type ParameterValue struct {
	Duration *Duration `parser:"@Duration |" json:"duration,omitempty"`
	Value    *Value    `parser:"@@" json:"value,omitempty"`
}
