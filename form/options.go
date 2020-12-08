// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

// An Field is used to define the behavior and appearance of a data form.
type Field func(*Data)

// Title sets a form's title.
func Title(s string) Field {
	return func(data *Data) {
		data.title = s
	}
}

// Instructions adds new textual instructions to the form.
func Instructions(s string) Field {
	return func(data *Data) {
		data.instructions = s
	}
}

var (
	// Result marks a form as the result type.
	// For more information see TypeResult.
	Result Field = result
)

var (
	result Field = func(data *Data) {
		data.typ = TypeResult
	}
)

// A Option is used to define the behavior and appearance of a form field.
type Option func(*field)

var (
	// Required flags the field as required in order for the form to be considered
	// valid.
	Required Option = required
)

var (
	required Option = func(f *field) {
		f.required = true
	}
)

// Desc provides a natural-language description of the field, intended for
// presentation in a user-agent (e.g., as a "tool-tip", help button, or
// explanatory text provided near the field).
// Desc should not contain newlines (the \n and \r characters), since layout is
// the responsibility of a user agent.
// However, it does nothing to prevent them from being added.
func Desc(s string) Option {
	return func(f *field) {
		f.desc = s
	}
}

// Value defines the default value for the field.
// Fields of type ListMulti, JidMulti, TextMulti, and Hidden may contain more
// than one Value; all other field types will only use the first Value.
func Value(s string) Option {
	return func(f *field) {
		f.value = append(f.value, s)
	}
}

// Label defines a human-readable name for the field.
func Label(s string) Option {
	return func(f *field) {
		f.label = s
	}
}

// ListItem adds a list item with the provided label and value.
// It has no effect on any non-list field type.
func ListItem(label, value string) Option {
	return func(f *field) {
		f.option = append(f.option, fieldOpt{
			Label: label,
			Value: value,
		})
	}
}

func getFieldOpts(f *field, o ...Option) {
	for _, opt := range o {
		opt(f)
	}
	return
}
