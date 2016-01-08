.PHONEY: test
test:
	GO15VENDOREXPERIMENT=1 go test -cover $$(GO15VENDOREXPERIMENT=1 go list ./... | grep -v '/vendor/')

deps.svg: *.go
	(   echo "digraph G {"; \
	GO15VENDOREXPERIMENT=1 go list -f '{{range .Imports}}{{printf "\t%q -> %q;\n" $$.ImportPath .}}{{end}}' \
		$$(GO15VENDOREXPERIMENT=1 go list -f '{{join .Deps " "}}' .) .; \
	echo "}"; \
	) | dot -Tsvg -o $@
