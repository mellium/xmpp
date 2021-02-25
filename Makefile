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


generate: disco/categories.go

disco/categories.go: disco/disco.go
	go generate ./disco
