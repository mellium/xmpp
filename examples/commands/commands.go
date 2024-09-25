// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The commands command lists and executes ad-hoc commands.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"log"
	"mellium.im/xmlstream"
	"os"
	"strings"
	"text/tabwriter"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/oob"

	"github.com/rivo/tview"
)

func main() {
	addr := os.Getenv("XMPP_ADDR")
	pass := os.Getenv("XMPP_PASS")

	myJID, err := jid.Parse(addr)
	if err != nil {
		log.Fatalf("error parsing XMPP_ADDR %q as a JID: %v", addr, err)
	}

	if len(os.Args) < 2 {
		log.Fatalf("not enough arguments, missing JID to query")
	}

	theirAddr := os.Args[1]
	theirJID, err := jid.Parse(theirAddr)
	if err != nil {
		log.Fatalf("error parsing argument %q as a JID: %v", theirAddr, err)
	}

	session, err := xmpp.DialClientSession(
		context.TODO(),
		myJID,
		xmpp.StartTLS(&tls.Config{
			ServerName: myJID.Domain().String(),
			MinVersion: tls.VersionTLS12,
		}),
		xmpp.SASL("", pass, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1, sasl.Plain),
		xmpp.BindResource(),
	)
	if err != nil {
		log.Fatalf("error logging in as %s: %v", myJID, err)
	}

	go func() {
		err := session.Serve(nil)
		if err != nil {
			log.Fatalf("session ended with error: %v", err)
		}
	}()

	cmdIter := commands.Fetch(context.TODO(), theirJID, session)

	if len(os.Args) > 2 {
		err = executeCommand(context.TODO(), os.Args[2], cmdIter, theirJID, session)
		if err != nil {
			log.Fatalf("error executing %s: %v", os.Args[2], err)
		}
		return
	}

	err = listCommands(cmdIter, theirJID, session)
	if err != nil {
		log.Fatalf("error listing ad-hoc commands: %v", err)
	}
}

func executeCommand(ctx context.Context, cmdName string, cmdIter commands.Iter, theirJID jid.JID, session *xmpp.Session) error {
	var cmd commands.Command
	for cmdIter.Next() {
		cmd = cmdIter.Command()
		if cmd.Node == cmdName {
			break
		}
	}
	if err := cmdIter.Err(); err != nil {
		return err
	}
	err := cmdIter.Close()
	if err != nil {
		return err
	}

	if cmd.Node == "" {
		return fmt.Errorf("no command %s advertised by %v", cmdName, theirJID)
	}

	var newAction commands.Actions
	err = cmd.ForEach(ctx, nil, session, func(resp commands.Response, payload xml.TokenReader) (commands.Command, xml.TokenReader, error) {
		var payloads []xml.TokenReader
		var actions commands.Actions
		var foundForm bool
		iter := xmlstream.NewIter(payload)
		for iter.Next() {
			start, inner := iter.Current()
			if start == nil {
				continue
			}

			d := xml.NewTokenDecoder(xmlstream.Wrap(inner, *start))
			// Pop the start element to put the decoder in the correct state.
			_, err = d.Token()
			if err != nil {
				return commands.Command{}, nil, err
			}
			switch {
			case start.Name.Space == commands.NS && start.Name.Local == "note":
				err = handleNote(d, start)
				if err != nil {
					return commands.Command{}, nil, err
				}
			case start.Name.Space == oob.NS:
				err = handleOOB(d, start)
				if err != nil {
					return commands.Command{}, nil, err
				}
			case start.Name.Space == form.NS:
				foundForm = true
				var formData form.Data
				err := d.DecodeElement(&formData, start)
				if err != nil {
					return commands.Command{}, nil, err
				}

				var newPayload xml.TokenReader
				newAction, newPayload, err = handleForm(formData, actions, resp)
				if err != nil {
					return commands.Command{}, nil, err
				}
				if newPayload != nil {
					payloads = append(payloads, newPayload)
				}
				switch newAction {
				case commands.Prev, commands.Next, commands.Complete:
					actions &^= commands.Execute
					actions |= newAction << 3
				}
			case start.Name.Space == commands.NS && start.Name.Local == "actions":
				// Just decode the actions, they will be displayed at the end.
				err := d.DecodeElement(&actions, start)
				if err != nil {
					return commands.Command{}, nil, err
				}
			default:
				return commands.Command{}, nil, fmt.Errorf("unsupported payload %v", start.Name)
			}
		}
		if err := iter.Err(); err != nil {
			return commands.Command{}, nil, err
		}

		// Actions are not part of the normal ordered flow of child elements, so we
		// ask the user for further input last (regardless of where the actions
		// appeared in the XML). However, if we displayed a form (or "forms") after we
		// encountered the actions we've already shown the actions to the user in the
		// form interface, so don't ask for them again.
		if !foundForm || (newAction != 0 && newAction&^commands.Execute == 0) {
			newAction, err = handleActions(actions, resp, session)
			if err != nil {
				return commands.Command{}, nil, fmt.Errorf("error handling actions: %v", err)
			}
			switch newAction {
			case commands.Prev, commands.Next, commands.Complete:
				actions &^= commands.Execute
				actions |= newAction << 3
			}
		}
		allPayloads := xmlstream.MultiReader(payloads...)
		switch actions >> 3 {
		case commands.Prev:
			return resp.Prev(), allPayloads, nil
		case commands.Next:
			return resp.Next(), allPayloads, nil
		case commands.Complete:
			return resp.Complete(), allPayloads, nil
		case 0:
			return resp.Cancel(), allPayloads, nil
		}
		return commands.Command{}, nil, nil
	})
	if err != nil {
		return fmt.Errorf("error executing multi-stage command: %w", err)
	}
	return nil
}

func handleActions(actions commands.Actions, resp commands.Response, session *xmpp.Session) (commands.Actions, error) {
	// Don't show a cancel button if it's the only action.
	if actions&^commands.Execute == 0 {
		return 0, nil
	}
	fmt.Print("Please enter one of the following actions: ")
	var actionStrings []string
	fmt.Print("[cancel")
	for i := commands.Actions(1); i <= commands.Complete; i <<= 1 {
		action := actions & i
		if action == 0 {
			continue
		}
		fmt.Print(", ")
		if action == ((actions & commands.Execute) >> 3) {
			fmt.Print("*")
		}
		fmt.Printf("%s", action)
		actionStrings = append(actionStrings, action.String())
	}
	fmt.Println("]")
	var text string
inputloop:
	for {
		fmt.Printf("\n> ")
		reader := bufio.NewReader(os.Stdin)
		text, _ = reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		// If there's a default action and we left it blank or used the string
		// "execute" go ahead and terminate the loop.
		// Alternatively, always terminate if we type "cancel".
		if ((text == "" || text == "execute") && actions>>3 != 0) || text == "cancel" {
			break
		}
		for _, actionName := range actionStrings {
			if text == actionName {
				break inputloop
			}
		}
	}

	switch text {
	case "", "execute":
		return actions >> 3, nil
	case "prev":
		return commands.Prev, nil
	case "next":
		return commands.Next, nil
	case "complete", "submit":
		return commands.Complete, nil
	}
	return 0, nil
}

func handleOOB(d *xml.Decoder, start *xml.StartElement) error {
	var oobURL oob.Data
	err := d.DecodeElement(&oobURL, start)
	if err != nil {
		return err
	}
	fmt.Printf("Description: %s\nURL: %s\n", oobURL.Desc, oobURL.URL)
	return nil
}

func handleNote(d *xml.Decoder, start *xml.StartElement) error {
	var note commands.Note
	err := d.DecodeElement(&note, start)
	if err != nil {
		return err
	}
	var colorEsc string
	switch note.Type {
	case commands.NoteInfo:
		// Blue
		colorEsc = "\033[34m"
	case commands.NoteWarn:
		// Yellow
		colorEsc = "\033[33m"
	case commands.NoteError:
		// Red
		colorEsc = "\033[31m"
	}
	// Print with the given color then reset formatting.
	fmt.Printf("%sNote: %s\033[0m\n", colorEsc, note.Value)
	return nil
}

func handleForm(formData form.Data, actions commands.Actions, resp commands.Response) (commands.Actions, xml.TokenReader, error) {
	app := tview.NewApplication()

	title := "Data Form"
	if t := formData.Title(); t != "" {
		title = t
	}
	box := tview.NewForm()
	box.SetBorder(true).SetTitle(title)
	if instructions := formData.Instructions(); instructions != "" {
		box.AddFormItem(tview.NewTextView().SetText(instructions).SetWrap(true))
	}
	formData.ForFields(func(field form.FieldData) {
		switch field.Type {
		case form.TypeBoolean:
			// TODO: changed func/required
			def, _ := formData.GetBool(field.Var)
			box.AddCheckbox(field.Label, def, func(checked bool) {
				_, err := formData.Set(field.Var, checked)
				if err != nil {
					log.Printf("error setting bool form field %s: %v", field.Var, err)
				}
			})
		case form.TypeFixed:
			box.AddFormItem(tview.NewTextView().SetText(field.Label))
			// TODO: will this just work? it's on the form already right?
		//case form.TypeHidden:
		//box.AddButton("Hidden: "+field.Label, nil)
		case form.TypeJIDMulti:
			jids, _ := formData.GetJIDs(field.Var)
			opts := make([]string, 0, len(jids))
			for _, j := range jids {
				opts = append(opts, j.String())
			}
			box.AddDropDown(field.Label, opts, 0, func(option string, optionIndex int) {
				j, err := jid.Parse(option)
				if err != nil {
					log.Printf("error parsing jid-multi value for field %s: %v", field.Var, err)
					return
				}
				_, err = formData.Set(field.Var, j)
				if err != nil {
					log.Printf("error setting jid-multi form field %s: %v", field.Var, err)
				}
			})
		case form.TypeJID:
			j, _ := formData.GetJID(field.Var)
			box.AddInputField(field.Label, j.String(), 20, func(textToCheck string, _ rune) bool {
				_, err := jid.Parse(textToCheck)
				return err != nil
			}, func(text string) {
				j := jid.MustParse(text)
				_, err := formData.Set(field.Var, j)
				if err != nil {
					log.Printf("error setting jid form field %s: %v", field.Var, err)
				}
			})
		case form.TypeListMulti, form.TypeList:
			// TODO: multi select list?
			opts, _ := formData.GetStrings(field.Var)
			box.AddDropDown(field.Label, opts, 0, func(option string, optionIndex int) {
				_, err := formData.Set(field.Var, option)
				if err != nil {
					log.Printf("error setting list or list-multi form field %s: %v", field.Var, err)
				}
			})
		case form.TypeTextMulti, form.TypeText:
			// TODO: multi line text, max lengths, etc.
			t, _ := formData.GetString(field.Var)
			box.AddInputField(field.Label, t, 20, nil, func(text string) {
				_, err := formData.Set(field.Var, text)
				if err != nil {
					log.Printf("error setting text or text-multi form field %s: %v", field.Var, err)
				}
			})
		case form.TypeTextPrivate:
			// TODO: multi line text, max lengths, etc.
			t, _ := formData.GetString(field.Var)
			box.AddPasswordField(field.Label, t, 20, '*', func(text string) {
				_, err := formData.Set(field.Var, text)
				if err != nil {
					log.Printf("error setting password form field %s: %v", field.Var, err)
				}
			})
		}
	})
	var action commands.Actions
	var submit xml.TokenReader
	if actions&commands.Prev == commands.Prev {
		box.AddButton("Previous", func() {
			submit, _ = formData.Submit()
			app.Stop()
			action = commands.Prev
		})
	}
	if actions&commands.Next == commands.Next {
		box.AddButton("Next", func() {
			submit, _ = formData.Submit()
			app.Stop()
			action = commands.Next
		})
	}
	if actions&commands.Complete == commands.Complete {
		box.AddButton("Submit", func() {
			submit, _ = formData.Submit()
			app.Stop()
			action = commands.Complete
		})
	}
	box.AddButton("Cancel", func() {
		app.Stop()
		action = 0
	})
	// If no actions were found always show a "submit" button.
	// The actions will then be shown later in the CLI view.
	if actions == 0 {
		box.AddButton("Submit", func() {
			submit, _ = formData.Submit()
			app.Stop()
			action = commands.Complete << 3
		})
	}

	err := app.SetRoot(box, true).EnableMouse(true).Run()
	if err != nil {
		return action, submit, err
	}

	return action, submit, nil
}

func listCommands(cmdIter commands.Iter, theirJID jid.JID, session *xmpp.Session) error {
	/* #nosec */
	defer cmdIter.Close()
	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	for cmdIter.Next() {
		cmd := cmdIter.Command()
		fmt.Fprintf(tabWriter, "%s\t%s\n", cmd.Node, cmd.Name)
	}
	err := cmdIter.Err()
	if err != nil {
		return err
	}
	return tabWriter.Flush()
}
