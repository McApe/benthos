package query

import (
	"errors"
	"testing"

	"github.com/Jeffail/benthos/v3/lib/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArithmetic(t *testing.T) {
	type easyMsg struct {
		content string
		meta    map[string]string
	}

	arithmetic := func(fns []Function, ops []ArithmeticOperator) Function {
		t.Helper()
		fn, err := NewArithmeticExpression(fns, ops)
		require.NoError(t, err)
		return fn
	}
	function := func(name string, args ...interface{}) Function {
		t.Helper()
		fn, err := InitFunction(name, args...)
		require.NoError(t, err)
		return fn
	}

	tests := map[string]struct {
		input    Function
		output   interface{}
		err      error
		messages []easyMsg
		index    int
	}{
		"compare string to int": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction("foo"),
					NewLiteralFunction(int64(5)),
				},
				[]ArithmeticOperator{
					ArithmeticNeq,
				},
			),
			output: true,
		},
		"dont divide by zero": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(5)),
					NewLiteralFunction(int64(0)),
				},
				[]ArithmeticOperator{
					ArithmeticDiv,
				},
			),
			err: errors.New("attempted to divide by zero"),
		},
		"dont divide by zero 2": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(5)),
					NewLiteralFunction(int64(0)),
				},
				[]ArithmeticOperator{
					ArithmeticMod,
				},
			),
			err: errors.New("attempted to divide by zero"),
		},
		"compare string to null": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction("foo"),
					NewLiteralFunction(nil),
				},
				[]ArithmeticOperator{
					ArithmeticNeq,
				},
			),
			output: true,
		},
		"compare string to int 2": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(5)),
					NewLiteralFunction("foo"),
				},
				[]ArithmeticOperator{
					ArithmeticNeq,
				},
			),
			output: true,
		},
		"compare string to null 2": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(nil),
					NewLiteralFunction("foo"),
				},
				[]ArithmeticOperator{
					ArithmeticNeq,
				},
			),
			output: true,
		},
		"add strings": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction("foo"),
					NewLiteralFunction("bar"),
					NewLiteralFunction("baz"),
				},
				[]ArithmeticOperator{
					ArithmeticAdd,
					ArithmeticAdd,
				},
			),
			output: `foobarbaz`,
		},
		"comparisons with not": {
			input: arithmetic(
				[]Function{
					Not(NewLiteralFunction(true)),
					NewLiteralFunction(false),
				},
				[]ArithmeticOperator{
					ArithmeticOr,
				},
			),
			output: false,
		},
		"comparisons with not 2": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(false),
					Not(NewLiteralFunction(false)),
				},
				[]ArithmeticOperator{
					ArithmeticOr,
				},
			),
			output: true,
		},
		"mod two ints": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(5)),
					NewLiteralFunction(int64(2)),
				},
				[]ArithmeticOperator{
					ArithmeticMod,
				},
			),
			output: int64(1),
		},
		"number comparisons": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(5.0),
					NewLiteralFunction(5.0),
				},
				[]ArithmeticOperator{
					ArithmeticNeq,
				},
			),
			output: false,
		},
		"comparisons": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(true),
					NewLiteralFunction(false),
					NewLiteralFunction(true),
					NewLiteralFunction(false),
				},
				[]ArithmeticOperator{
					ArithmeticAnd,
					ArithmeticOr,
					ArithmeticAnd,
				},
			),
			output: false,
		},
		"comparisons 2": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(false),
					NewLiteralFunction(true),
					NewLiteralFunction(true),
					NewLiteralFunction(false),
				},
				[]ArithmeticOperator{
					ArithmeticOr,
					ArithmeticAnd,
					ArithmeticOr,
				},
			),
			output: true,
		},
		"comparisons 3": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(true),
					NewLiteralFunction(false),
					NewLiteralFunction(true),
				},
				[]ArithmeticOperator{
					ArithmeticOr,
					ArithmeticAnd,
				},
			),
			output: true,
		},
		"err comparison": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction("not a number"),
					NewLiteralFunction(int64(0)),
				},
				[]ArithmeticOperator{
					ArithmeticGt,
				},
			),
			err: errors.New("expected string value, found number: 0"),
		},
		"numbers comparison": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(float64(15)),
					NewLiteralFunction(uint64(0)),
				},
				[]ArithmeticOperator{
					ArithmeticGt,
				},
			),
			output: true,
		},
		"numbers comparison 2": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(0)),
					NewLiteralFunction(float64(15)),
				},
				[]ArithmeticOperator{
					ArithmeticGt,
				},
			),
			output: false,
		},
		"numbers comparison 3": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(uint64(15)),
					NewLiteralFunction(int64(15)),
				},
				[]ArithmeticOperator{
					ArithmeticGte,
				},
			),
			output: true,
		},
		"numbers comparison 4": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(uint64(15)),
					NewLiteralFunction(float64(15)),
				},
				[]ArithmeticOperator{
					ArithmeticLte,
				},
			),
			output: true,
		},
		"numbers comparison 5": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(15)),
					NewLiteralFunction(float64(15)),
				},
				[]ArithmeticOperator{
					ArithmeticLt,
				},
			),
			output: false,
		},
		"and exit early": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(false),
					arithmetic(
						[]Function{
							NewLiteralFunction("not a number"),
							NewLiteralFunction(int64(0)),
						},
						[]ArithmeticOperator{
							ArithmeticGt,
						},
					),
				},
				[]ArithmeticOperator{
					ArithmeticAnd,
				},
			),
			output: false,
		},
		"and second exit early": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(true),
					NewLiteralFunction(false),
					arithmetic(
						[]Function{
							NewLiteralFunction("not a number"),
							NewLiteralFunction(int64(0)),
						},
						[]ArithmeticOperator{
							ArithmeticGt,
						},
					),
				},
				[]ArithmeticOperator{
					ArithmeticAnd,
					ArithmeticAnd,
				},
			),
			output: false,
		},
		"or exit early": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(true),
					arithmetic(
						[]Function{
							NewLiteralFunction("not a number"),
							NewLiteralFunction(int64(0)),
						},
						[]ArithmeticOperator{
							ArithmeticGt,
						},
					),
				},
				[]ArithmeticOperator{
					ArithmeticOr,
				},
			),
			output: true,
		},
		"or second exit early": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(false),
					NewLiteralFunction(true),
					arithmetic(
						[]Function{
							NewLiteralFunction("not a number"),
							NewLiteralFunction(int64(0)),
						},
						[]ArithmeticOperator{
							ArithmeticGt,
						},
					),
				},
				[]ArithmeticOperator{
					ArithmeticOr,
					ArithmeticOr,
				},
			),
			output: true,
		},
		"multiply and additions of ints 3": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(2)),
					NewLiteralFunction(int64(3)),
					NewLiteralFunction(float64(2)),
					NewLiteralFunction(uint64(1)),
					NewLiteralFunction(uint64(3)),
				},
				[]ArithmeticOperator{
					ArithmeticAdd,
					ArithmeticMul,
					ArithmeticAdd,
					ArithmeticMul,
				},
			),
			output: float64(11),
		},
		"division and subtractions of ints": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(6)),
					NewLiteralFunction(int64(6)),
					NewLiteralFunction(float64(2)),
					NewLiteralFunction(uint64(1)),
				},
				[]ArithmeticOperator{
					ArithmeticSub,
					ArithmeticDiv,
					ArithmeticAdd,
				},
			),
			output: float64(4),
		},
		"coalesce json": {
			input: arithmetic(
				[]Function{
					function("json", "foo"),
					function("json", "bar"),
				},
				[]ArithmeticOperator{
					ArithmeticPipe,
				},
			),
			output: `from_bar`,
			messages: []easyMsg{
				{content: `{"foo":null,"bar":"from_bar"}`},
			},
		},
		"coalesce json 2": {
			input: arithmetic(
				[]Function{
					function("json", "foo"),
					NewLiteralFunction("not this"),
				},
				[]ArithmeticOperator{
					ArithmeticPipe,
				},
			),
			output: `from_foo`,
			messages: []easyMsg{
				{content: `{"foo":"from_foo"}`},
			},
		},
		"coalesce delete unmapped": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(Delete(nil)),
					NewLiteralFunction(Nothing(nil)),
					NewLiteralFunction("this"),
				},
				[]ArithmeticOperator{
					ArithmeticPipe,
					ArithmeticPipe,
				},
			),
			output: `this`,
		},
		"compare maps": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(map[string]interface{}{
						"foo": "bar",
					}),
					NewLiteralFunction(map[string]interface{}{
						"foo": "bar",
					}),
				},
				[]ArithmeticOperator{
					ArithmeticEq,
				},
			),
			output: true,
		},
		"compare maps neg": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(map[string]interface{}{
						"foo": "bar",
					}),
					NewLiteralFunction(map[string]interface{}{
						"foo": "baz",
					}),
				},
				[]ArithmeticOperator{
					ArithmeticNeq,
				},
			),
			output: true,
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

			res, err := test.input.Exec(FunctionContext{
				Index:    test.index,
				MsgBatch: msg,
			})
			if test.err != nil {
				require.EqualError(t, err, test.err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.output, res)
			}
		})
	}
}

func TestArithmeticTargets(t *testing.T) {
	arithmetic := func(fns []Function, ops []ArithmeticOperator) Function {
		t.Helper()
		fn, err := NewArithmeticExpression(fns, ops)
		require.NoError(t, err)
		return fn
	}
	function := func(name string, args ...interface{}) Function {
		t.Helper()
		fn, err := InitFunction(name, args...)
		require.NoError(t, err)
		return fn
	}

	tests := map[string]struct {
		input  Function
		output []TargetPath
	}{
		"no targets": {
			input: arithmetic(
				[]Function{
					NewLiteralFunction(int64(5)),
					NewLiteralFunction("bar"),
				},
				[]ArithmeticOperator{
					ArithmeticAdd,
				},
			),
			output: nil,
		},
		"coalesced targets": {
			input: arithmetic(
				[]Function{
					function("meta", "foo"),
					function("var", "bar"),
				},
				[]ArithmeticOperator{
					ArithmeticPipe,
				},
			),
			output: []TargetPath{
				NewTargetPath(TargetMetadata, "foo"),
				NewTargetPath(TargetVariable, "bar"),
			},
		},
		"mix of function types": {
			input: arithmetic(
				[]Function{
					function("meta", "buz"),
					NewLiteralFunction(int64(5)),
					function("json", "foo.bar"),
					NewLiteralFunction("bar"),
					NewFieldFunction("qux.quz"),
				},
				[]ArithmeticOperator{
					ArithmeticEq,
					ArithmeticAdd,
					ArithmeticMul,
					ArithmeticGt,
				},
			),
			output: []TargetPath{
				NewTargetPath(TargetMetadata, "buz"),
				NewTargetPath(TargetValue, "foo", "bar"),
				NewTargetPath(TargetValue, "qux", "quz"),
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
