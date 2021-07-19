// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form_test

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/jid"
)

var (
	_ xml.Marshaler       = (*form.Data)(nil)
	_ xmlstream.Marshaler = (*form.Data)(nil)
	_ xmlstream.WriterTo  = (*form.Data)(nil)
	_ xml.Unmarshaler     = (*form.Data)(nil)
)

var submissionTestCases = [...]struct {
	Data     *form.Data
	Ok       bool
	Expected string
}{
	0: {
		// A nil form can still be submitted.
		Expected: `<x xmlns="jabber:x:data" type="submit"></x>`,
		Ok:       true,
	},
	1: {
		// A form should not include unset fields that are not required.
		Data:     form.New(form.Boolean("boolvar")),
		Expected: `<x xmlns="jabber:x:data" type="submit"></x>`,
		Ok:       true,
	},
	2: {
		// A form should return ok is false for fields that are unset but required.
		Data:     form.New(form.Boolean("boolvar", form.Required)),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="boolean" var="boolvar"><required></required><value>false</value></field></x>`,
		Ok:       false,
	},
	3: {
		// A form should return a default for unset but required fields that have
		// one.
		Data:     form.New(form.Boolean("boolvar", form.Value("true"), form.Required)),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="boolean" var="boolvar"><required></required><value>true</value></field></x>`,
		Ok:       true,
	},
	4: {
		// Bools should also support "1" and "0".
		Data:     form.New(form.Boolean("boolvar", form.Value("0"))),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="boolean" var="boolvar"><value>false</value></field></x>`,
		Ok:       true,
	},
	5: {
		// A form should not return fixed fields.
		Data:     form.New(form.Fixed()),
		Expected: `<x xmlns="jabber:x:data" type="submit"></x>`,
		Ok:       true,
	},
	6: {
		// A form should return the first default for single line text.
		Data:     form.New(form.Text("textvar", form.Value("one"), form.Value("two"))),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="text-single" var="textvar"><value>one</value></field></x>`,
		Ok:       true,
	},
	7: {
		// A form should return the first valid default for a single JID.
		Data:     form.New(form.JID("jidvar", form.Value("//"), form.Value("two"), form.Value("three"))),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="jid-single" var="jidvar"><value>two</value></field></x>`,
		Ok:       true,
	},
	8: {
		// TODO: implement and test dedup as well?
		// A form should return only valid defaults for JIDs.
		Data:     form.New(form.JIDMulti("jidvar", form.Value("//"), form.Value("two"), form.Value("three"))),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="jid-multi" var="jidvar"><value>two</value><value>three</value></field></x>`,
		Ok:       true,
	},
	9: {
		Data:     form.New(form.TextMulti("textvar", form.Value("one"), form.Value("two"), form.Value("three"))),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="text-multi" var="textvar"><value>one</value><value>two</value><value>three</value></field></x>`,
		Ok:       true,
	},
	10: {
		Data:     form.New(form.ListMulti("listvar", form.Value("one"), form.Value("two"), form.Value("three"))),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="list-multi" var="listvar"><value>one</value><value>two</value><value>three</value></field></x>`,
		Ok:       true,
	},
	11: {
		Data: func() *form.Data {
			data := form.New(form.TextMulti("textvar", form.Value("one"), form.Value("two"), form.Value("three")))
			data.ForFields(func(f form.FieldData) {
				text, _ := data.Get(f.Var)
				data.Set(f.Var, text.(string)+"a")
			})

			return data
		}(),
		Expected: `<x xmlns="jabber:x:data" type="submit"><field type="text-multi" var="textvar"><value>one</value><value>two</value><value>threea</value></field></x>`,
		Ok:       true,
	},
}

func TestSubmit(t *testing.T) {
	for i, tc := range submissionTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			submission, ok := tc.Data.Submit()
			var b bytes.Buffer
			e := xml.NewEncoder(&b)
			_, err := xmlstream.Copy(e, submission)
			if err != nil {
				t.Fatalf("error copying submission: %v", err)
			}
			err = e.Flush()
			if err != nil {
				t.Fatalf("error flushing encoder: %v", err)
			}
			if s := b.String(); s != tc.Expected {
				t.Errorf("wrong XML on marshal %d:\nwant=%s,\n got=%s", i, tc.Expected, s)
			}
			if ok != tc.Ok {
				t.Errorf("wrong value for ok: want=%t, got=%t", tc.Ok, ok)
			}
		})
	}
}

var marshalTestCases = [...]struct {
	Data     *form.Data
	Expected string
}{
	0: {
		Data:     form.Cancel("oops", "do\nstuff"),
		Expected: `<x xmlns="jabber:x:data" type="cancel"><title>oops</title><instructions>do</instructions><instructions>stuff</instructions></x>`,
	},
	1: {
		Data:     form.New(),
		Expected: `<x xmlns="jabber:x:data" type="form"></x>`,
	},
	2: {
		Data:     form.New(form.Title("1\n2\r3\r\n4\n\r5"), form.Instructions("6\r7\r\n8\n\r9")),
		Expected: `<x xmlns="jabber:x:data" type="form"><title>1 2 3 4 5</title><instructions>6</instructions><instructions>7</instructions><instructions>8</instructions><instructions>9</instructions></x>`,
	},
	3: {
		Data: form.New(
			form.Boolean("boolvar", form.Required, form.Desc("desc"), form.Value("a"), form.Value("true"), form.Value("false"), form.ListItem("", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="boolean" var="boolvar"><desc>desc</desc><required></required><value>true</value></field></x>`,
	},
	4: {
		Data: form.New(
			form.Fixed(form.Value("fixed"), form.ListItem("", "item"), form.Label("lab")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="fixed" label="lab"><value>fixed</value></field></x>`,
	},
	5: {
		Data: form.New(
			form.Hidden("hid", form.Value("h"), form.ListItem("", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="hidden" var="hid"><value>h</value></field></x>`,
	},
	6: {
		Data: form.New(
			form.JIDMulti("j", form.Value("//"), form.Value("jid@example.net"), form.Value("example.org"), form.ListItem("", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="jid-multi" var="j"><value>jid@example.net</value><value>example.org</value></field></x>`,
	},
	7: {
		Data: form.New(
			form.JID("j", form.Value(""), form.Value("//"), form.Value("jid@example.net"), form.Value("example.org"), form.ListItem("", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="jid-single" var="j"><value>jid@example.net</value></field></x>`,
	},
	8: {
		// TODO: make sure option/value are unique
		// TODO: are labels required?
		// TODO: See also the *** note in the XEP after text-single description
		Data: form.New(
			form.ListMulti("l", form.Value("one"), form.Value("two"),
				form.ListItem("label", "item"), form.ListItem("", "2")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="list-multi" var="l"><value>one</value><value>two</value><option label="label"><value>item</value></option><option label=""><value>2</value></option></field></x>`,
	},
	9: {
		Data: form.New(
			form.List("l", form.Value("one"), form.Value("two"),
				form.ListItem("label", "item"), form.ListItem("", "2")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="list-single" var="l"><value>one</value><option label="label"><value>item</value></option><option label=""><value>2</value></option></field></x>`,
	},
	10: {
		// TODO: should multiline values be split into multiple <value/> elements?
		Data: form.New(
			form.TextMulti("t", form.Value("one"), form.Value("two"), form.ListItem("label", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="text-multi" var="t"><value>one</value><value>two</value></field></x>`,
	},
	11: {
		Data: form.New(
			form.TextPrivate("t", form.Value("one"), form.Value("two"), form.ListItem("label", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="text-private" var="t"><value>one</value></field></x>`,
	},
	12: {
		Data: form.New(
			form.Text("t", form.Value("one"), form.Value("two"), form.ListItem("label", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="text-single" var="t"><value>one</value></field></x>`,
	},
	13: {
		Data: form.New(
			form.Text("t", form.Value("one"), form.Value("two"), form.ListItem("label", "item")),
			form.Boolean("b", form.Required),
		),
		Expected: `<x xmlns="jabber:x:data" type="form"><field type="text-single" var="t"><value>one</value></field><field type="boolean" var="b"><required></required></field></x>`,
	},
	14: {
		Data: form.New(
			form.Result,
			form.Text("t", form.Value("one"), form.Value("two"), form.ListItem("label", "item")),
		),
		Expected: `<x xmlns="jabber:x:data" type="result"><field type="text-single" var="t"><value>one</value></field></x>`,
	},
}

func TestMarshal(t *testing.T) {
	for i, tc := range marshalTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var b []byte
			var err error
			// Marshal twice to make sure we're not actually consuming something from
			// the slices in TokenReader.
			for i := 0; i < 2; i++ {
				b, err = xml.Marshal(tc.Data)
				if err != nil {
					t.Fatalf("error marshaling %d: %v", i, err)
				}
				if string(b) != tc.Expected {
					t.Errorf("wrong XML on marshal %d:\nwant=%s,\n got=%s", i, tc.Expected, b)
				}
			}

			data := &form.Data{}
			err = xml.Unmarshal(b, data)
			if err != nil {
				t.Fatalf("error unmarshaling: %v", err)
			}
			b, err = xml.Marshal(data)
			if err != nil {
				t.Fatalf("error remarshaling: %v", err)
			}
			if string(b) != tc.Expected {
				t.Errorf("wrong XML after remarshal:\nwant=%s,\n got=%s", tc.Expected, b)
			}
		})
	}
}

func TestSetGet(t *testing.T) {
	const (
		notExist        = "novar"
		boolVar         = "boolvar"
		jidVar          = "jidvar"
		jidsVar         = "jidsvar"
		listVar         = "listvar"
		multilistVar    = "multilistvar"
		multitextDefVar = "multitextdefvar"
	)
	data := form.New(
		form.Boolean(boolVar),
		form.JID(jidVar),
		form.JIDMulti(jidsVar),
		form.List(listVar),
		form.ListMulti(multilistVar),
		form.Fixed(),
		form.TextMulti(multitextDefVar, form.Value("one"), form.Value("two"), form.Value("three")),
	)
	t.Run("validate", func(t *testing.T) {
		ok, err := data.Set(boolVar, "wrong")
		if ok {
			t.Errorf("did not expect ok %s", boolVar)
		}
		if err == nil {
			t.Errorf("expected error on invalid %s", boolVar)
		}
		ok, err = data.Set(jidVar, "wrong")
		if ok {
			t.Errorf("did not expect ok %s", jidVar)
		}
		if err == nil {
			t.Errorf("expected error on invalid %s", jidVar)
		}
		ok, err = data.Set(jidsVar, "wrong")
		if ok {
			t.Errorf("did not expect ok %s", jidsVar)
		}
		if err == nil {
			t.Errorf("expected error on invalid %s", jidsVar)
		}
		ok, err = data.Set(listVar, 1)
		if ok {
			t.Errorf("did not expect ok %s", listVar)
		}
		if err == nil {
			t.Errorf("expected error on invalid %s", listVar)
		}
		ok, err = data.Set(multilistVar, 1)
		if ok {
			t.Errorf("did not expect ok %s", multilistVar)
		}
		if err == nil {
			t.Errorf("expected error on invalid %s", multilistVar)
		}
		ok, err = data.Set("", "test")
		if ok {
			t.Error("did not expect ok on fixed")
		}
		if err == nil {
			t.Error("expected error on set fixed")
		}
	})
	t.Run("strings", func(t *testing.T) {
		v, ok := data.GetString(multitextDefVar)
		if !ok || v != "one\ntwo\nthree" {
			t.Errorf("expected ok for %s before set got %v, %t", multilistVar, v, ok)
		}
		_, ok = data.GetString(listVar)
		if ok {
			t.Errorf("expected not ok for %s before set", listVar)
		}
		ok, err := data.Set(listVar, "one")
		if !ok {
			t.Errorf("did not expect ok to be false for %s", listVar)
		}
		if err != nil {
			t.Errorf("did not expect error when setting %s, got: %v", listVar, err)
		}
		_, ok = data.GetString(listVar)
		if !ok {
			t.Errorf("expected string for %s to be ok", listVar)
		}
		_, ok = data.GetStrings(multilistVar)
		if ok {
			t.Errorf("expected not ok for %s before set", multilistVar)
		}
		ok, err = data.Set(multilistVar, []string{"one", "two"})
		if !ok {
			t.Errorf("did not expect ok to be false for %s", multilistVar)
		}
		if err != nil {
			t.Errorf("did not expect error when setting %s, got: %v", multilistVar, err)
		}
		_, ok = data.GetString(multilistVar)
		if ok {
			t.Errorf("expected string for %s to not be ok", multilistVar)
		}
		_, ok = data.GetStrings(multilistVar)
		if !ok {
			t.Errorf("expected string slice for %s to be ok", multilistVar)
		}
	})
	t.Run("multijid", func(t *testing.T) {
		const varName = jidsVar
		v, ok := data.Get(varName)
		if ok {
			t.Errorf("expected getting %s to not be ok before it is set got %v", varName, v)
		}
		j, ok := data.GetJIDs(varName)
		if ok {
			t.Errorf("expected getting %s jids to not be ok before it is set got %v", varName, j)
		}
		ok, err := data.Set(varName, []jid.JID{jid.MustParse("example.net")})
		if !ok {
			t.Errorf("did not expect ok to be false for %s", varName)
		}
		if err != nil {
			t.Errorf("did not expect error when setting %s, got: %v", varName, err)
		}
		v, ok = data.Get(varName)
		if !ok {
			t.Errorf("expected getting %s to be ok after it was set", varName)
		}
		if _, ok = v.([]jid.JID); !ok {
			t.Errorf("expected ok for %s", varName)
		}
		_, ok = data.GetJIDs(varName)
		if !ok {
			t.Errorf("expected getting %s jids to be ok after it was set", varName)
		}
	})
	t.Run("jid", func(t *testing.T) {
		const varName = jidVar
		v, ok := data.Get(varName)
		if ok {
			t.Errorf("expected getting %s to not be ok before it is set got %v", varName, v)
		}
		j, ok := data.GetJID(varName)
		if ok {
			t.Errorf("expected getting %s jid to not be ok before it is set got %v", varName, j)
		}
		ok, err := data.Set(varName, jid.MustParse("example.net"))
		if !ok {
			t.Errorf("did not expect ok to be false for %s", varName)
		}
		if err != nil {
			t.Errorf("did not expect error when setting %s, got: %v", varName, err)
		}
		v, ok = data.Get(varName)
		if !ok {
			t.Errorf("expected getting %s to be ok after it was set", varName)
		}
		if _, ok = v.(jid.JID); !ok {
			t.Errorf("expected ok for %s", varName)
		}
		_, ok = data.GetJID(varName)
		if !ok {
			t.Errorf("expected getting %s jid to be ok after it was set", varName)
		}
	})
	t.Run("bool", func(t *testing.T) {
		const varName = boolVar
		v, ok := data.Get(varName)
		if ok {
			t.Errorf("expected getting %s to not be ok before it is set got %v", varName, v)
		}
		b, ok := data.GetBool(varName)
		if ok {
			t.Errorf("expected getting %s bool to not be ok before it is set got %t", varName, b)
		}
		ok, err := data.Set(varName, true)
		if !ok {
			t.Errorf("did not expect ok to be false for %s", varName)
		}
		if err != nil {
			t.Errorf("did not expect error when setting %s, got: %v", varName, err)
		}
		v, ok = data.Get(varName)
		if !ok {
			t.Errorf("expected getting %s to be ok after it was set", varName)
		}
		if b, ok = v.(bool); !ok || !b {
			t.Errorf("expected value of %s to be true got %v", varName, v)
		}
		b, ok = data.GetBool(varName)
		if !ok || !b {
			t.Errorf("expected getting %s bool to be true, ok after it was set got %t", varName, b)
		}
	})
	// Setting to an var that does not exist always works.
	t.Run("new variable", func(t *testing.T) {
		const varName = notExist
		v, ok := data.Get(varName)
		if ok {
			t.Errorf("expected getting %s to not be ok before it is set got %v", varName, v)
		}
		b, ok := data.GetBool(varName)
		if ok {
			t.Errorf("expected getting %s bool to not be ok before it is set got %t", varName, b)
		}
		ok, err := data.Set(varName, false)
		if ok {
			t.Errorf("did not expect ok to be true for %s", varName)
		}
		if err != nil {
			t.Errorf("did not expect error when setting %s, got: %v", varName, err)
		}
		v, ok = data.Get(varName)
		if !ok {
			t.Errorf("expected getting %s to be ok after it was set", varName)
		}
		if b, ok = v.(bool); !ok || b {
			t.Errorf("expected value of %s to be false got %v", varName, v)
		}
		b, ok = data.GetBool(varName)
		if !ok || b {
			t.Errorf("expected getting %s bool to be false, ok after it was set got %t", varName, b)
		}
	})
}

func TestUnmarshalChardata(t *testing.T) {
	const formData = `
<x xmlns="jabber:x:data" type="submit">
	<field type="boolean" var="foo">
		<required></required><value>true</value>
	</field>
</x>`
	data := &form.Data{}
	err := xml.Unmarshal([]byte(formData), data)
	if err != nil {
		t.Fatalf("error unmarshaling: %v", err)
	}
	if data.Len() != 1 {
		t.Errorf("wrong length: want=1, got=%d", data.Len())
	}
	if b, ok := data.GetBool("foo"); !ok || !b {
		t.Errorf("expected form field 'foo' to be set, got %t, %t", b, ok)
	}
	if v, ok := data.Raw("foo"); !ok || len(v) == 0 || v[0] != "true" {
		t.Errorf("expected form field 'foo' to have raw value, got %s, %t", v, ok)
	}
	if v, ok := data.Raw("test"); v != nil || ok {
		t.Errorf("did not expect raw values for unkonwn key, got %v, %t", v, ok)
	}
}

func TestUnmarshalInvalidToken(t *testing.T) {
	const formData = `<x xmlns="jabber:x:data"><!-- Not allowed --></x>`
	data := &form.Data{}
	err := xml.Unmarshal([]byte(formData), data)
	if data.Len() != 0 {
		t.Errorf("wrong length: want=0, got=%d", data.Len())
	}
	if err == nil {
		t.Fatalf("expected error when unmarshaling disallowed token type")
	}
}

func TestNilLen(t *testing.T) {
	var data *form.Data
	if data.Len() != 0 {
		t.Errorf("wrong length: want=0, got=%d", data.Len())
	}
	if v, ok := data.Raw("test"); v != nil || ok {
		t.Errorf("did not expect raw values, got %v, %t", v, ok)
	}
}
