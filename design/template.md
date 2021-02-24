# Title

**Author(s):** Your name <email@example.net>  
**Last updated:** 2020-04-06  
**Discussion:** https://mellium.im/issue/{issue number}

## Abstract

A sentence or two explaining *what* is being proposed.


## Terminology

This section includes any terminology that needs to be defined to understand the
rest of the document. This SHOULD include the latest text from [BCP 14]:

    The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL
    NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED",
    "MAY", and "OPTIONAL" in this document are to be interpreted as
    described in BCP 14 [RFC2119] [RFC8174] when, and only when, they
    appear in all capitals, as shown here.

[BCP 14]: https://tools.ietf.org/html/bcp14


## Background

A longer description explaining what is being proposed in more detail, but also
why it is needed and the history of the issue or the feature in Mellium if
applicable, as well as any necessary background information required to fully
understand and evaluate the proposal.


## Requirements

- This will likely be a bulleted list
- Each requirement should be met by the proposal in the next section


## Proposal

The actual proposal.
This should include the entire public API and a summary of what this will mean
for compatibility and maintenance overhead going forward.
For example, it might include the various types, functions and methods in a
fenced code block.
This is a good place to go ahead and think about your documentation and how you
will explain to users what your types, methods, and functions do and how they
should use them.
For example:

```
// Cmd is an external command being prepared or run.
type Cmd struct {
	*exec.Cmd
	…
}

	// New creates a new, unstarted, command.
	func New(ctx context.Context, name string, opts ...Option) (*Cmd, error)

	// Close kills the command if it is still running and cleans up any
	// temporary resources that were created.
	func (cmd *Cmd) Close() error

	// ConfigDir returns the temporary directory used to store config files.
	func (cmd *Cmd) ConfigDir() string

	// Dial attempts to connect to the server by dialing localhost and then
	// negotiating a stream with the location set to the domainpart of j and the
	// origin set to j.
	func (cmd *Cmd) Dial(ctx context.Context, j jid.JID, t *testing.T, features ...xmpp.StreamFeature) (*xmpp.Session, error)
```

## Open Issues

A list of known issues that still need to be addressed.
This section may be omitted if there are no open issues.
