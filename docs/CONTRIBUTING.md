# Contributing

Thanks for wanting to contribute to Mellium! Before submitting a patch, please
read our [Code of Conduct].


## Filing Issues

Bugs and feature requests can be started by opening an [issue][issues] (unless
it is a sensitive security issue, in which case keep reading).
Always open an issue before creating a pull request unless the PR is trivial,
all PRs should be linked to an issue and should generally only contain a single
logical change.

Don't forget to check the issue tracker ([including closed issues]) and [pull
requests] for existing issues and changes before you start work.
Once you file an issue or find an existing issue, make sure to mention that
you're working on the problem and outline your plans so that someone else
doesn't duplicate your work.

If you're not sure where to begin, grab any of the issues labeled [`good first
issue`], and if you need help with any of this, don't be afraid to ask!

Security sensitive issues should be reported directly to the project maintainer
by emailing [`security@mellium.im`] for more information see [SECURITY.md].


## Creating Patches

When you create your commit, be sure to follow convention for the commit message
and code formatting.

  - Format all code with `go fmt`
  - Write documentation comments for any new public identifiers
  - Write tests for your code
  - Follow Go best practices
  - Write a detailed commit message
  - Submit a pull request and wait for review

Commit messages should start with the name of the Go package being modified, or
the string "all" if it affects the entire module, followed by a colon.
The rest of the first line should be a short description of how it modifies the
project, for example, the following is a good first line for a commit message to
the `dial` package:

    dial: fix flaky tests

After the first line should be a blank line, followed by a paragraph or so
describing the change in more detail.
This provides context for the commit and should be written in full sentences.
Do not use Markdown, HTML, or other formatting in your commit messages.
You may also include benchmarks and other data that provides context and shows
why your commit should be merged, the Go [benchstat] tool may be helpful for
this.

For example, a good full commit message might be:

    dial: fix flaky tests

    Previously a DNS request might have been made for A or AAAA records
    depending on what networks were available. Tests expected AAAA requests
    so they would fail on machines that only had IPv4 networking.


## Pull Requests

Once your pull request is submitted, you will hear back from a maintainer within
5 days.
If you haven't heard back by then, feel free to ping the PR to move it back to
the top of peoples inboxes.

To update an existing pull request you may add more commits on top of the first
commit or amend and push the existing commit.
Once your change is accepted your reviewer may ask you to rebase your branch
on top of the base branch and squash it into a single commit that can be merged,
or they may handle this for you.


## Review

All patches must be reviewed by a maintainer before being merged.
Don't be discouraged if the maintainer asks questions or requests changes, even
for simple patches.
This is perfectly normal, and means that the maintainers are interested in your
change and that it stands a good chance of being merged after the changes are
complete!


## Sign your work

All commits must be signed before they can be accepted. Your signature
indicates that you have the right to contribute the work and that it can be
contributed as open source. The exact rules can be viewed at
[developercertificate.org], or in the file [DCO].
Your signature also indicates that you have read and agree to the license
statement in the next section.

To add your signature, add a line like the following to the end of your commit
message with your name and email:

    Signed-off-by: Andrew Aguecheek <aaguecheek@example.net>

You can add this line easily using Git by committing with `git commit -s`.
If you forget to add a signature to a commit, quickly add it to the latest
commit with `git commit --amend -s --no-edit`.


## License

The package may be used under the terms of the BSD 2-Clause License a copy of
which may be found in the file "[LICENSE]".

Unless you explicitly state otherwise, any contribution submitted for inclusion
in the work by you shall be licensed as above, without any additional terms or
conditions.


[issues]: https://github.com/mellium/xmpp/issues
[including closed issues]: https://github.com/mellium/xmpp/issues?q=is%3Aissue
[pull requests]: https://github.com/mellium/xmpp/pulls?q=is%3Apr
[`good first issue`]: https://github.com/mellium/xmpp/labels/good%20first%20issue
[`security@mellium.im`]: mailto:security@mellium.im
[benchstat]: https://godoc.org/golang.org/x/perf/cmd/benchstat
[developercertificate.org]: https://developercertificate.org/
[DCO]: ./DCO
[LICENSE]: ../LICENSE
[SECURITY.md]: ./SECURITY.md
[Code of Conduct]: ./CODE_OF_CONDUCT.md
