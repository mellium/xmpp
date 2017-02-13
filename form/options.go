// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package form

// An Option is used to define the behavior and appearance of a data form.
type Option func(*Data)

// Title sets a form's title.
func Title(s string) Option {
	return func(data *Data) {
		data.title.Text = s
	}
}

// Instructions adds new textual instructions to the form.
// Multiple uses of the option result in multiple sets of instructions.
func Instructions(s string) Option {
	return func(data *Data) {
		data.children = append(data.children, instructions{Text: s})
	}
}

func getOpts(data *Data, o ...Option) {
	for _, f := range o {
		f(data)
	}
	return
}

// A FieldOption is used to define the behavior and appearance of a form field.
type FieldOption func(*field)

var (
	// Required flags the field as required in order for the form to be considered
	// valid.
	Required FieldOption = required
)

var (
	required FieldOption = func(f *field) {
		f.Required = &struct{}{}
	}
)

// Desc provides a natural-language description of the field, intended for
// presentation in a user-agent (e.g., as a "tool-tip", help button, or
// explanatory text provided near the field).
// Desc should not contain newlines (the \n and \r characters), since layout is
// the responsibility of a user agent.
// However, it does nothing to prevent them from being added.
func Desc(s string) FieldOption {
	return func(f *field) {
		f.Desc = s
	}
}

// Value defines the default value for the field (according to the
// form-processing entity) in a data form of type "form", the data provided by a
// form-submitting entity in a data form of type "submit", or a data result in a
// data form of type "result".
// Fields of type ListMulti, JidMulti, TextMulti, and Hidden may contain more
// than one Value; all other field types will only use the first Value.
func Value(s string) FieldOption {
	return func(f *field) {
		f.Value = append(f.Value, s)
	}
}

// ListOption is one of the values in a list.
// It has no effect on any non-list field type.
func ListOption(s string) FieldOption {
	return func(f *field) {
		f.Option = append(f.Option, fieldopt{
			Value: s,
		})
	}
}

func getFieldOpts(f *field, o ...FieldOption) {
	for _, opt := range o {
		opt(f)
	}
	return
}
