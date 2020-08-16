package query

import (
	"errors"
	"testing"

	"github.com/Jeffail/benthos/v3/lib/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressions(t *testing.T) {
	type easyMsg struct {
		content string
		meta    map[string]string
	}

	mustFunc := func(fn Function, err error) Function {
		t.Helper()
		require.NoError(t, err)
		return fn
	}

	tests := map[string]struct {
		input    Function
		value    *interface{}
		output   interface{}
		err      error
		messages []easyMsg
		index    int
	}{
		"if false": {
			input: NewIfFunction(
				mustFunc(NewArithmeticExpression(
					[]Function{
						NewLiteralFunction(int64(10)),
						NewLiteralFunction(int64(20)),
					},
					[]ArithmeticOperator{
						ArithmeticGt,
					},
				)),
				NewLiteralFunction("foo"),
				nil,
			),
			output: Nothing(nil),
		},
		"if false else": {
			input: NewIfFunction(
				mustFunc(NewArithmeticExpression(
					[]Function{
						NewLiteralFunction(int64(10)),
						NewLiteralFunction(int64(20)),
					},
					[]ArithmeticOperator{
						ArithmeticGt,
					},
				)),
				NewLiteralFunction("foo"),
				NewLiteralFunction("bar"),
			),
			output: "bar",
		},
		"if true": {
			input: NewIfFunction(
				mustFunc(NewArithmeticExpression(
					[]Function{
						NewLiteralFunction(int64(10)),
						NewLiteralFunction(int64(20)),
					},
					[]ArithmeticOperator{
						ArithmeticLt,
					},
				)),
				NewLiteralFunction("foo"),
				NewLiteralFunction(Nothing(nil)),
			),
			output: "foo",
		},
		"if query fails": {
			input: NewIfFunction(
				NewVarFunction("doesnt exist"),
				NewLiteralFunction("foo"),
				NewLiteralFunction("bar"),
			),
			err: errors.New("failed to check if condition: variables were undefined"),
		},
		"match context fails": {
			input: NewMatchFunction(
				NewVarFunction("doesnt exist"),
				NewMatchCase(NewLiteralFunction(true), NewLiteralFunction("foo")),
			),
			err: errors.New("variables were undefined"),
		},
		"match first case fails": {
			input: NewMatchFunction(
				NewLiteralFunction("context"),
				NewMatchCase(NewVarFunction("doesnt exist"), NewLiteralFunction("foo")),
				NewMatchCase(NewLiteralFunction(true), NewLiteralFunction("bar")),
			),
			err: errors.New("failed to check match case 0: variables were undefined"),
		},
		"match second case fails": {
			input: NewMatchFunction(
				NewLiteralFunction("context"),
				NewMatchCase(NewLiteralFunction(true), NewLiteralFunction("bar")),
				NewMatchCase(NewVarFunction("doesnt exist"), NewLiteralFunction("foo")),
			),
			output: "bar",
		},
		"match context empty": {
			input: NewMatchFunction(
				nil,
				NewMatchCase(NewLiteralFunction(true), NewFieldFunction("")),
			),
			value: func() *interface{} {
				var v interface{} = "context"
				return &v
			}(),
			output: "context",
		},
		"match context": {
			input: NewMatchFunction(
				NewLiteralFunction("context"),
				NewMatchCase(NewLiteralFunction(true), NewFieldFunction("")),
			),
			output: "context",
		},
		"match context all fail": {
			input: NewMatchFunction(
				NewLiteralFunction("context"),
				NewMatchCase(NewLiteralFunction(false), NewLiteralFunction("foo")),
				NewMatchCase(NewLiteralFunction(false), NewLiteralFunction("bar")),
			),
			output: Nothing(nil),
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			msg := message.New(nil)
			for _, m := range test.messages {
				part := message.NewPart([]byte(m.content))
				if m.meta != nil {
					for k, v := range m.meta {
						part.Metadata().Set(k, v)
					}
				}
				msg.Append(part)
			}

			for i := 0; i < 10; i++ {
				res, err := test.input.Exec(FunctionContext{
					Value:    test.value,
					Maps:     map[string]Function{},
					Index:    test.index,
					MsgBatch: msg,
				})
				if test.err != nil {
					require.EqualError(t, err, test.err.Error())
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, test.output, res)
			}

			// Ensure nothing changed
			for i, m := range test.messages {
				doc, err := msg.Get(i).JSON()
				if err == nil {
					msg.Get(i).SetJSON(doc)
				}
				assert.Equal(t, m.content, string(msg.Get(i).Get()))
			}
		})
	}
}

func TestExpressionTargets(t *testing.T) {
	mustFunc := func(fn Function, err error) Function {
		t.Helper()
		require.NoError(t, err)
		return fn
	}

	tests := map[string]struct {
		input  Function
		output []TargetPath
	}{
		"if query path": {
			input: NewIfFunction(
				mustFunc(InitFunction("json", "foo.bar")),
				NewLiteralFunction("foo"),
				mustFunc(InitFunction("var", "baz")),
			),
			output: []TargetPath{
				NewTargetPath(TargetValue, "foo", "bar"),
				NewTargetPath(TargetVariable, "baz"),
			},
		},
		"match empty context": {
			input: NewMatchFunction(
				nil,
				NewMatchCase(
					NewFieldFunction("foo"),
					NewFieldFunction("bar"),
				),
				NewMatchCase(
					NewFieldFunction("baz"),
					NewFieldFunction("buz"),
				),
			),
			output: []TargetPath{
				NewTargetPath(TargetValue, "foo"),
				NewTargetPath(TargetValue, "bar"),
				NewTargetPath(TargetValue, "baz"),
				NewTargetPath(TargetValue, "buz"),
			},
		},
		"match meta context": {
			input: NewMatchFunction(
				mustFunc(InitFunction("meta", "foo")),
				NewMatchCase(
					mustFunc(InitFunction("meta", "bar")),
					NewFieldFunction("baz"),
				),
				NewMatchCase(
					NewFieldFunction("buz"),
					NewLiteralFunction("qux"),
				),
			),
			output: []TargetPath{
				NewTargetPath(TargetMetadata, "bar"),
				NewTargetPath(TargetValue, "baz"),
				NewTargetPath(TargetValue, "buz"),
				NewTargetPath(TargetMetadata, "foo"),
			},
		},
		"match value context": {
			input: NewMatchFunction(
				NewFieldFunction("foo.bar"),
				NewMatchCase(
					mustFunc(InitFunction("meta", "bar")),
					NewFieldFunction("baz.buz"),
				),
				NewMatchCase(
					NewFieldFunction("qux.quz"),
					NewLiteralFunction("quack"),
				),
			),
			output: []TargetPath{
				NewTargetPath(TargetMetadata, "bar"),
				NewTargetPath(TargetValue, "foo", "bar", "baz", "buz"),
				NewTargetPath(TargetValue, "foo", "bar", "qux", "quz"),
				NewTargetPath(TargetValue, "foo", "bar"),
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			res := test.input.QueryTargets()
			assert.Equal(t, test.output, res)
		})
	}
}
