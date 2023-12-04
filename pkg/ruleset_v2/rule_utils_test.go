package ruleset_v2

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseFuncCall(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected struct {
			funcName string
			params   []string
			err      error
		}
	}{
		{
			name:  "Normal case, one param",
			input: `one("baz")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "one", params: []string{"baz"}, err: nil},
		},
		{
			name:  "Normal case, one param, extra space in function call",
			input: `two("baz" )`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "two", params: []string{"baz"}, err: nil},
		},
		{
			name:  "Normal case, one param, extra space in param",
			input: `three("baz ")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "three", params: []string{"baz "}, err: nil},
		},
		{
			name:  "Space in front of function",
			input: ` three("baz")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "three", params: []string{"baz"}, err: nil},
		},
		{
			name:  "Normal case, two params",
			input: `foobar("baz", "qux")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "foobar", params: []string{"baz", "qux"}, err: nil},
		},
		{
			name:  "Normal case, two params, no spaces between param comma",
			input: `foobar("baz","qux")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "foobar", params: []string{"baz", "qux"}, err: nil},
		},
		{
			name:  "Escaped parenthesis",
			input: `testFunc("hello\(world", "anotherParam")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "testFunc", params: []string{`hello(world`, "anotherParam"}, err: nil},
		},
		{
			name:  "Escaped quote",
			input: `testFunc("hello\"world", "anotherParam")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "testFunc", params: []string{`hello"world`, "anotherParam"}, err: nil},
		},
		{
			name:  "Two Escaped quote",
			input: `testFunc("hello: \"world\"", "anotherParam")`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "testFunc", params: []string{`hello: "world"`, "anotherParam"}, err: nil},
		},
		{
			name:  "No parameters",
			input: `emptyFunc()`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "emptyFunc", params: []string{}, err: nil},
		},
		{
			name:  "Invalid format",
			input: `invalidFunc`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "", params: nil, err: errors.New("invalid function call format")},
		},
		{
			name:  "Invalid format 2",
			input: `invalidFunc "foo", "bar"`,
			expected: struct {
				funcName string
				params   []string
				err      error
			}{funcName: "", params: nil, err: errors.New("invalid function call format")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			funcName, params, err := parseFuncCall(tc.input)
			if funcName != tc.expected.funcName || !reflect.DeepEqual(params, tc.expected.params) || (err != nil && tc.expected.err != nil && err.Error() != tc.expected.err.Error()) {
				//if funcName != tc.expected.funcName || (err != nil && tc.expected.err != nil && err.Error() != tc.expected.err.Error()) {
				t.Errorf("Test %s failed: got (%s, %v, %v), want (%s, %v, %v)", tc.name, funcName, params, err, tc.expected.funcName, tc.expected.params, tc.expected.err)
			}
		})
	}
}
