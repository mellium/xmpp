// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The genpubsub command creates pubsub error condition and feature types.
package main // import "mellium.im/xmpp/internal/genpubsub"

import (
	"bytes"
	"encoding/xml"
	"flag"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/go/packages"
)

const tmpl = `// Code generated by "genpubsub{{if gt (len .Args) 0}} {{end}}{{.Args}}"; DO NOT EDIT.

package {{.Pkg}}

import (
	"encoding/xml"
)

// Condition is the underlying cause of a pubsub error.
type Condition uint32

{{- $last := "" }}
// Valid pubsub Conditions.
const (
CondNone Condition = iota
{{- range .Conditions }}
	Cond{{ . | ident }} // {{ . }}
	{{- $last = . }}
{{- end }}
)

// UnmarshalXML implements xml.Unmarshaler.
func (c *Condition) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for cond := CondNone; cond <= Cond{{ $last | ident }}; cond++ {
		if cond.String() == start.Name.Local {
			*c = cond
			break
		}
	}
	return d.Skip()
}

// Feature is a specific pubsub feature that may be reported in an error as
// being unsupported.
type Feature uint32

// Valid pubsub Features.
const (
{{- with index .Features 0 }}
	Feature{{ . | ident }} Feature = iota // {{ . }}
{{- end }}
{{- range slice .Features 1 }}
	Feature{{ . | ident }} // {{ . }}
{{- end }}
)

// SubType represents the state of a particular subscription.
type SubType uint8

// A list of possible subscription types.
const (
{{- with index .SubTypes 0 }}
	Sub{{ . | ident }} SubType = iota // {{ . }}
{{- end }}
{{- range slice .SubTypes 1 }}
	Sub{{ . | ident }} // {{ . }}
{{- end }}
)`

type errorsSchema struct {
	XMLName xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
	Comment string   `xml:"annotation>documentation"`
	Element []struct {
		Name  string `xml:"name,attr"`
		Value []struct {
			Value string `xml:"value,attr"`
		} `xml:"complexType>simpleContent>extension>attribute>simpleType>restriction>enumeration"`
	} `xml:"http://www.w3.org/2001/XMLSchema element"`
}

type eventSchema struct {
	XMLName xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
	Comment string   `xml:"annotation>documentation"`
	Element []struct {
		Name  string `xml:"name,attr"`
		Attrs []struct {
			Name string `xml:"name,attr"`
			Enum []struct {
				Value string `xml:"value,attr"`
			} `xml:"simpleType>restriction>enumeration"`
		} `xml:"complexType>simpleContent>extension>attribute"`
	} `xml:"http://www.w3.org/2001/XMLSchema element"`
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("genpubsub: ")
	var (
		outFile = "conditions.go"
		regURL  = `https://xmpp.org/schemas/`
		tmpDir  = os.TempDir()
		noFmt   bool
	)

	flag.StringVar(&outFile, "filename", outFile, "filename to generate")
	flag.StringVar(&tmpDir, "tmp", tmpDir, "A temporary directory to downlaod files to")
	flag.StringVar(&regURL, "schema", regURL, "A link to the pubsub schema directory")
	flag.BoolVar(&noFmt, "nofmt", noFmt, "Disables code formatting")
	flag.Parse()

	errsPath := regURL + "pubsub-errors.xsd"
	errsFile, err := openOrDownload(errsPath, tmpDir)
	if err != nil {
		log.Fatalf("error downloading (or opening) %s in %s: %v", errsPath, tmpDir, err)
	}
	/* #nosec */
	defer errsFile.Close()

	eventPath := regURL + "pubsub-event.xsd"
	eventFile, err := openOrDownload(eventPath, tmpDir)
	if err != nil {
		log.Fatalf("error downloading (or opening) %s in %s: %v", eventPath, tmpDir, err)
	}
	/* #nosec */
	defer eventFile.Close()

	s := errorsSchema{}
	d := xml.NewDecoder(errsFile)
	var errsStart xml.StartElement
	for {
		var ok bool
		tok, err := d.Token()
		if err != nil {
			log.Fatalf("error popping err schema token: %v", err)
		}
		errsStart, ok = tok.(xml.StartElement)
		if ok && errsStart.Name.Local == "schema" {
			break
		}
	}
	if err = d.DecodeElement(&s, &errsStart); err != nil {
		log.Fatalf("error decoding err schema: %v", err)
	}

	event := eventSchema{}
	eventDecoder := xml.NewDecoder(eventFile)
	var eventStart xml.StartElement
	for {
		var ok bool
		tok, err := eventDecoder.Token()
		if err != nil {
			log.Fatalf("error popping event schema token: %v", err)
		}
		eventStart, ok = tok.(xml.StartElement)
		if ok && errsStart.Name.Local == "schema" {
			break
		}
	}
	if err = eventDecoder.DecodeElement(&event, &eventStart); err != nil {
		log.Fatalf("error decoding event schema: %v", err)
	}

	// Flatten the schema values to make them easier to iterate over in the
	// template.
	var (
		conditions []string
		features   []string
		subtypes   []string
	)
	for _, e := range s.Element {
		conditions = append(conditions, e.Name)
		for _, f := range e.Value {
			features = append(features, f.Value)
		}
	}
	for _, e := range event.Element {
		for _, a := range e.Attrs {
			if a.Name == "subscription" {
				for _, t := range a.Enum {
					subtypes = append(subtypes, t.Value)
				}
			}
		}
	}

	pkgs, err := packages.Load(nil, ".")
	if err != nil {
		log.Fatalf("error loading package: %v", err)
	}
	pkg := pkgs[0]

	parsedTmpl, err := template.New("out").Funcs(map[string]interface{}{
		"ident": func(s string) string {
			s = strings.TrimSpace(s)
			var buf strings.Builder
			var nextCap bool
			for i, b := range s {
				if i == 0 || nextCap {
					buf.WriteRune(unicode.ToUpper(b))
					nextCap = false
					continue
				}
				if b == '-' {
					nextCap = true
					continue
				}
				buf.WriteRune(b)
			}

			// Return the identifier with a few special case replacements.
			return strings.NewReplacer(
				"Jid", "JID",
				"Nodeid", "NodeID",
				"Subid", "SubID",
				"ConfigurationRequired", "ConfigRequired",
				"PresenceSubscriptionRequired", "PresenceRequired",
				"Ids", "IDs",
			).Replace(buf.String())
		},
	}).Parse(tmpl)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}

	var buf bytes.Buffer
	errsFile, err = os.Create(outFile)
	if err != nil {
		log.Fatalf("error creating file %q: %v", outFile, err)
	}
	err = parsedTmpl.Execute(&buf, struct {
		Args       string
		Pkg        string
		Features   []string
		Conditions []string
		SubTypes   []string
	}{
		Args:       strings.Join(os.Args[1:], " "),
		Pkg:        pkg.Name,
		Features:   features,
		Conditions: conditions,
		SubTypes:   subtypes,
	})
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	if noFmt {
		_, err = io.Copy(errsFile, &buf)
		if err != nil {
			log.Fatalf("error writing file: %v", err)
		}
	} else {
		fmtBuf, err := format.Source(buf.Bytes())
		if err != nil {
			log.Fatalf("error formatting source: %v", err)
		}
		_, err = io.Copy(errsFile, bytes.NewReader(fmtBuf))
		if err != nil {
			log.Fatalf("error writing file: %v", err)
		}
	}
}

// opens the provided schema URL (downloading it if it doesn't exist).
func openOrDownload(catURL, tmpDir string) (*os.File, error) {
	schemaXML := filepath.Join(tmpDir, filepath.Base(catURL))
	/* #nosec */
	fd, err := os.Open(schemaXML)
	if err != nil {
		/* #nosec */
		fd, err = os.Create(schemaXML)
		if err != nil {
			return nil, err
		}
		// If we couldn't open it for reading, attempt to download it.

		/* #nosec */
		resp, err := http.Get(catURL)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fd, resp.Body)
		if err != nil {
			return nil, err
		}
		/* #nosec */
		resp.Body.Close()
		_, err = fd.Seek(0, 0)
		if err != nil {
			return nil, err
		}
	}
	return fd, err
}
