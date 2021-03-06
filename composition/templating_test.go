package composition

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"io"
	"net/url"
	"testing"
)

func Test_Templating_Variables(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		data     map[string]interface{}
		template string
		expected string
	}{
		{
			data:     map[string]interface{}{},
			template: "xxx",
			expected: "xxx",
		},
		{
			data:     map[string]interface{}{},
			template: "",
			expected: "",
		},
		{
			data:     nil,
			template: "--§[foo]§--",
			expected: "----",
		},
		{
			data:     map[string]interface{}{"foo": "bar"},
			template: "§[foo]§",
			expected: "bar",
		},
		{
			data:     map[string]interface{}{"some": url.Values{"url": {"param"}}},
			template: "§[some.url]§",
			expected: "param",
		},
		{
			data:     map[string]interface{}{"foo": map[string]interface{}{"bar": "bazz"}},
			template: "§[foo.bar]§",
			expected: "bazz",
		},
		{
			data:     map[string]interface{}{"foo": map[string]interface{}{"bar": "bazz"}, "foo.bar": "overwrite"},
			template: "§[foo.bar]§",
			expected: "overwrite",
		},
		{
			data:     map[string]interface{}{"foo": map[string]interface{}{"bar": "bazz"}},
			template: "§[foo.bar.nothing]§",
			expected: "",
		},
		{
			data:     map[string]interface{}{"foo": "bar"},
			template: "§[ foo ]§",
			expected: "bar",
		},
		{
			data:     map[string]interface{}{"foo": "bar"},
			template: "xxx-§[foo]§-yyy",
			expected: "xxx-bar-yyy",
		},
		{
			data:     map[string]interface{}{"foo": "bar", "bli": "blub"},
			template: "xxx-§[foo]§-yyy-§[bli]§-zzz",
			expected: "xxx-bar-yyy-blub-zzz",
		},
		{
			data:     map[string]interface{}{},
			template: "xxx-§[not_existent_variable]§-yyy",
			expected: "xxx--yyy",
		},
		{
			data:     map[string]interface{}{},
			template: "xxx-]§-yyy",
			expected: "xxx-]§-yyy",
		},
	}

	for _, test := range tests {

		buf := bytes.NewBufferString("")
		err := executeTemplate(buf, test.template, test.data, nil)

		a.NoError(err)

		a.Equal(test.expected, buf.String())
	}
}

func Test_expandTemplateVars(t *testing.T) {
	a := assert.New(t)
	result, err := expandTemplateVars("§[foo]§", map[string]interface{}{"foo": "bar"})
	a.NoError(err)
	a.Equal("bar", result)
}

func Test_Templating_Includes(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		fragments   map[string]string
		template    string
		expected    string
		expectedErr error
	}{
		{
			fragments: map[string]string{"foo": "bar"},
			template:  "§[> foo]§",
			expected:  "bar",
		},
		{
			fragments: map[string]string{"foo": "bar"},
			template:  "§[>   foo   ]§",
			expected:  "bar",
		},
		{
			fragments: map[string]string{"foo": "bar"},
			template:  "xxx-§[> foo]§-yyy",
			expected:  "xxx-bar-yyy",
		},
		{
			fragments: map[string]string{"foo": "bar", "bli": "blub"},
			template:  "xxx-§[> foo]§-yyy-§[> bli]§-zzz",
			expected:  "xxx-bar-yyy-blub-zzz",
		},
		{
			fragments:   map[string]string{},
			template:    "xxx-§[> not_existent_fragment]§-yyy",
			expected:    "xxx-",
			expectedErr: errors.New("Fragment does not exist: not_existent_fragment"),
		},
		// Optional includes with alternative text
		{
			fragments: map[string]string{"foo": "bar"},
			template:  "xxx-§[#> foo]§ alternative text §[/foo]§-yyy",
			expected:  "xxx-bar-yyy",
		},
		{
			fragments: map[string]string{},
			template:  "xxx-§[#> foo]§ alternative text §[/foo]§-yyy",
			expected:  "xxx- alternative text -yyy",
		},
		{
			fragments:   map[string]string{},
			template:    "xxx-§[#> foo]§ alternative text §-yyy",
			expectedErr: errors.New("Fragment parsing error, missing ending block: §[/foo]§"),
		},
	}

	for _, test := range tests {
		buf := bytes.NewBufferString("")
		executeNestedFragment := func(nestedFragmentName string) error {
			if val, exist := test.fragments[nestedFragmentName]; exist {
				io.WriteString(buf, val)
				return nil
			}
			return errors.New("Fragment does not exist: " + nestedFragmentName)
		}
		err := executeTemplate(buf, test.template, nil, executeNestedFragment)

		if test.expectedErr == nil {
			a.NoError(err)
			a.Equal(test.expected, buf.String())
		} else {
			a.Equal(test.expectedErr, err)
		}

	}
}

func Test_Templating_ParsingErrors(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		template          string
		expectedErrString string
	}{
		{
			template:          "xxx-§[-yyy",
			expectedErrString: "Fragment parsing error, missing ending separator:",
		},
		{
			template:          "xxx-]§§[-yyy",
			expectedErrString: "Fragment parsing error, missing ending separator:",
		},
	}

	for _, test := range tests {
		buf := bytes.NewBufferString("")
		executeNestedFragment := func(nestedFragmentName string) error {
			return nil
		}
		err := executeTemplate(buf, test.template, map[string]interface{}{}, executeNestedFragment)
		a.Error(err)
		a.Contains(err.Error(), test.expectedErrString)
	}
}
