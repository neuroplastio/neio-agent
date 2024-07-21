package flowdsl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func TestStatements(t *testing.T) {
	type testCase struct {
		input    string
		expected Statement
	}

	testCases := []testCase{
		{
			input: `lock(key.Esc)`,
			expected: Statement{
				Expr: &ExpressionStatement{
					Identifier: "lock",
					Arguments: []Argument{
						{
							Usage: &UsageStatement{
								Usages: []string{
									"key.Esc",
								},
							},
						},
					},
				},
			},
		},
		{
			input: `tapHold(Esc, LeftShift+LeftAlt, 250ms)`,
			expected: Statement{
				Expr: &ExpressionStatement{
					Identifier: "tapHold",
					Arguments: []Argument{
						{
							Usage: &UsageStatement{
								Usages: []string{
									"Esc",
								},
							},
						},
						{
							Usage: &UsageStatement{
								Usages: []string{
									"LeftShift",
									"LeftAlt",
								},
							},
						},
						{
							Duration: ptr(Duration(250 * time.Millisecond)),
						},
					},
				},
			},
		},
		{
			input: `typeTest(0, key.2, false, true, "something")`,
			expected: Statement{
				Expr: &ExpressionStatement{
					Identifier: "typeTest",
					Arguments: []Argument{
						{
							Value: &Value{
								Number: ptr(Number("0")),
							},
						},
						{
							Usage: &UsageStatement{
								Usages: []string{
									"key.2",
								},
							},
						},
						{
							Value: &Value{
								Boolean: ptr(Boolean(false)),
							},
						},
						{
							Value: &Value{
								Boolean: ptr(Boolean(true)),
							},
						},
						{
							Value: &Value{
								String: ptr("something"),
							},
						},
					},
				},
			},
		},
		{
			input: `$layer.switch("base")`,
			expected: Statement{
				Expr: &ExpressionStatement{
					Identifier: "$layer.switch",
					Arguments: []Argument{
						{
							Value: &Value{
								String: ptr("base"),
							},
						},
					},
				},
			},
		},
		{
			input: `signal(null, $layer.unset("base"))`,
			expected: Statement{
				Expr: &ExpressionStatement{
					Identifier: "signal",
					Arguments: []Argument{
						{
							Value: &Value{},
						},
						{
							Expr: &ExpressionStatement{
								Identifier: "$layer.unset",
								Arguments: []Argument{
									{
										Value: &Value{
											String: ptr("base"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			actual, err := statementParser.ParseString("", tc.input, participle.Trace(buf))
			if !assert.NoError(t, err) {
				t.Log(buf.String())
				return
			}

			expectedJSON, err := json.Marshal(tc.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(actual)
			require.NoError(t, err)

			require.Equal(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestDeclarations(t *testing.T) {
	type testCase struct {
		input    string
		expected Declaration
	}

	testCases := []testCase{
		{
			input: `lock(action: Action)`,
			expected: Declaration{
				Identifier: "lock",
				Parameters: []Parameter{
					{
						Name: "action",
						Type: "Action",
					},
				},
			},
		},
		{
			input: `tapHold(onTap: Action, onHold: Action, delay: Duration = 250ms, tapDuration: Duration = 25ms)`,
			expected: Declaration{
				Identifier: "tapHold",
				Parameters: []Parameter{
					{
						Name: "onTap",
						Type: "Action",
					},
					{
						Name: "onHold",
						Type: "Action",
					},
					{
						Name: "delay",
						Type: "Duration",
						Default: &ParameterValue{
							Duration: ptr(Duration(250 * time.Millisecond)),
						},
					},
					{
						Name: "tapDuration",
						Type: "Duration",
						Default: &ParameterValue{
							Duration: ptr(Duration(25 * time.Millisecond)),
						},
					},
				},
			},
		},
		{
			input: `test(str: string = "val", num: number = 49, bool: boolean = false, dur: Duration = 100m, val: any = null)`,
			expected: Declaration{
				Identifier: "test",
				Parameters: []Parameter{
					{
						Name: "str",
						Type: "string",
						Default: &ParameterValue{
							Value: &Value{
								String: ptr("val"),
							},
						},
					},
					{
						Name: "num",
						Type: "number",
						Default: &ParameterValue{
							Value: &Value{
								Number: ptr(Number("49")),
							},
						},
					},
					{
						Name: "bool",
						Type: "boolean",
						Default: &ParameterValue{
							Value: &Value{
								Boolean: ptr(Boolean(false)),
							},
						},
					},
					{
						Name: "dur",
						Type: "Duration",
						Default: &ParameterValue{
							Duration: ptr(Duration(100 * time.Minute)),
						},
					},
					{
						Name: "val",
						Type: "any",
						Default: &ParameterValue{
							Value: &Value{},
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			actual, err := declarationParser.ParseString("", tc.input, participle.Trace(buf))
			if !assert.NoError(t, err) {
				t.Log(buf.String())
				return
			}

			expectedJSON, err := json.Marshal(tc.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(actual)
			require.NoError(t, err)

			require.Equal(t, string(expectedJSON), string(actualJSON))
		})
	}

}

func TestUsages(t *testing.T) {
	type testCase struct {
		input    string
		expected UsageStatement
	}

	testCases := []testCase{
		{
			input: "LeftAlt+btn.5",
			expected: UsageStatement{
				Usages: []string{
					"LeftAlt",
					"btn.5",
				},
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			actual, err := usageParser.ParseString("", tc.input, participle.Trace(buf))
			if !assert.NoError(t, err) {
				t.Log(buf.String())
				return
			}

			expectedJSON, err := json.Marshal(tc.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(actual)
			require.NoError(t, err)

			require.Equal(t, string(expectedJSON), string(actualJSON))
		})
	}

}
