.POSIX:
.SILENT:

CONTRIBUTORS: FORCE
	echo "// This is the official list of contributors for copyright purposes." > $@
	echo "//" >> $@
	echo "// If you see your name twice, please fix your commits or create a .mailmap" >> $@
	echo "// entry for yourself and regenerate this file by running make CONTRIBUTORS." >> $@
	echo "// For more info see https://www.git-scm.com/docs/git-check-mailmap" >> $@
	echo "" >> $@
	git --no-pager shortlog --summary --email | cut -f2- >> $@

FORCE:

styling_tests.json: styling/styling_test.go styling/export_test.go
	go test -tags export ./styling -args -export=$@
	mv styling/$@ .

GOFILES!=find . -name '*.go'
deps.svg: $(GOFILES)
	hash dot 2>/dev/null || (echo "No 'dot' found, please install graphviz" && exit 1)
	(   echo "digraph G {"; \
	go list -f '{{range .Imports}}{{printf "\t%q -> %q;\n" $$.ImportPath .}}{{end}}' \
		$$(go list -f '{{join .Deps " "}}' .) .; \
	echo "}"; \
	) | dot -Tsvg -o $@

######
##
## Code Gen
##
## Below this are shortcuts for generating files created with go generate.
## All files can be updated simply by running "go generate" but they are
## included here in make format for documentation purposes.
##
######

carbons/disco.go: carbons/carbons.go
	go generate ./carbons

receipts/disco.go: receipts/receipts.go
	go generate ./receipts

form/disco.go: form/doc.go
	go generate ./form

ping/disco.go: ping/ping.go
	go generate ./ping

oob/disco.go: oob/oob.go
	go generate ./disco

ibr2/disco.go: ibr2/doc.go
	go generate ./ibr2

muc/disco.go: muc/muc.go
	go generate -run="genfeature" ./muc

muc/affililiation_string.go: muc/types.go
	go generate -run="stringer" ./muc

paging/disco.go: paging/rsm.go
	go generate ./paging

forward/disco.go: forward/forward.go
	go generate ./forward

xtime/disco.go: xtime/time.go
	go generate ./xtime

jid/disco.go: jid/doc.go
	go generate ./jid

version/disco.go: version/version.go
	go generate ./version

commands/disco.go: commands/commands.go
	go generate -run="genfeature" ./commands

commands/actions_string.go: commands/actions.go
	go generate -run="stringer -type=Actions" ./commands

commands/notetype_string.go: commands/actions.go
	go generate -run="stringer -type=NoteType" ./commands

color/cvd_string.go: color/color.go
	go generate ./color

disco/categories.go: disco/disco.go
	go generate -run="gen.go" ./disco

disco/features.go: disco/disco.go
	go generate -run="genfeature" ./disco

styling/styling_string.go: styling/styling.go
	go generate -run="stringer" ./styling

styling/disco.go: styling/styling.go
	go generate -run="genfeature" ./styling

sessionstate_string.go: session.go
	go generate
