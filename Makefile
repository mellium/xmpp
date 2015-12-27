.PHONEY: test
test:
	GO15VENDOREXPERIMENT=1 go test $$(go list ./... | grep -v '/vendor/')

deps.svg: *.go
	(   echo "digraph G {"; \
	go list -f '{{range .Imports}}{{printf "\t%q -> %q;\n" $$.ImportPath .}}{{end}}' \
		$$(go list -f '{{join .Deps " "}}' .) .; \
	echo "}"; \
	) | dot -Tsvg -o $@
