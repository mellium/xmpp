// +build ignore

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"

	"mellium.im/xmpp/form"
)

func main() {
	b := new(bytes.Buffer)
	e := xml.NewEncoder(b)
	e.Indent("", "\t")
	f := form.New(
		form.Title("Title"),
		form.Instructions("Instructions to fill out the form!"),
		form.Boolean("bool", form.Required),
		form.Fixed(),
	)
	if err := e.Encode(f); err != nil {
		log.Fatal(err)
	}
	fmt.Println(b.String())

	d := xml.NewDecoder(conn)

	data := form.Data{}
	xml.Decode(&data)
}
