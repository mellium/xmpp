.PHONY: generate
.POSIX:
.SILENT:

CONTRIBUTORS:
	echo "// This is the official list of contributors for copyright purposes." > $@
	echo "//" >> $@
	echo "// If you see your name twice, please fix your commits or create a .mailmap" >> $@
	echo "// entry for yourself and regenerate this file by running make CONTRIBUTORS." >> $@
	echo "// For more info see https://www.git-scm.com/docs/git-check-mailmap" >> $@
	echo "" >> $@
	git --no-pager shortlog --summary --email | cut -f2- >> $@

generate: session.go color/cvd_string.go styling/styling_string.go disco/categories.go

color/cvd_string.go: color/color.go
	go generate ./color

disco/categories.go: disco/disco.go
	go generate ./disco

styling/styling_string.go: styling/styling.go
	go generate ./styling

sessionstate_string.go: session.go
	go generate
