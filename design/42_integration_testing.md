# Proposal: Create integration testing package

**Author(s):** Sam Whited  
**Last updated:** 2020-04-28  
**Discussion:** https://mellium.im/issue/42


## Abstract

An API is proposed for running and configuring XMPP servers for use in
integration tests.


## Background

The nature of an XMPP library requires integration tests.
Not only because a network protocol naturally needs external resources (like a
network and a server), but because the public Jabber network is built on
interoperability, and any XMPP library must be able to integrate with it.
Robust integration testing will require running various servers such as
[Prosody] and [Ejabberd] which will require different configuration,
certificates, and other resources for different tests.
To facilitate this a package should be written to create a temporary directory
with various config files and other resources and then run commands such as
`ejabberdctl` pointing to this directory.


[Prosody]: https://prosody.im/
[Ejabberd]: https://www.ejabberd.im/


## Requirements

- Ability to run integration tests locally without manually configuring servers
  or other tools
- Must be able to run one set of tests against multiple servers
- Ability to run multiple tests against the same server before shutting down the
  child process


## Proposal

The proposed API adds three new types, seven functions, and three methods to an
internal package that does not have to be covered by the compatibility promise:


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

// Option is used to configure a Cmd.
type Option func(cmd *Cmd) error

	// Args sets additional command line args to be passed to the command.
	func Args(f ...string) Option

	// Cert creates a private key and certificate with the given name.
	func Cert(name string) Option

	// Log configures the command to log output to the current testing.T.
	func Log() Option

	// LogXML configures the command to log sent and received XML to the current
	// testing.T.
	func LogXML() Option

	// TempFile creates a file in the commands temporary working directory.
	// After all configuration is complete it then calls f to populate the
	// config files.
	func TempFile(cfgFileName string, f func(*Cmd, io.Writer) error) Option

// SubtestRunner is the signature of a function that can be used to start
// subtests.
type SubtestRunner func(func(context.Context, *testing.T, *Cmd)) bool

	// Test starts a command and returns a function that runs f as a subtest
	// using t.Run. Multiple calls to the returned function will result in
	// uniquely named subtests. When all subtests have completed, the daemon is
	// stopped.
	func Test(ctx context.Context, name string, t *testing.T, opts ...Option) SubtestRunner
```

The main downside to this proposal is that `Test` is a higher-order function
that also returns a higher-order function.
This is confusing and results in some odd syntax such as:

```
integration.Test(
	context.Background(), t,
	integration.Cert("localhost"),
	…
)(func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	…
})
```

Unfortunately, this can't be helped.
If we were to put everything in one function (including the subtest function and
the variadic options) it becomes even worse:

```
integration.Test(
	context.Background(), t,
	func(ctx context.Context, t *testing.T, cmd *integration.Cmd) {
	 …
	},
	integration.Cert("localhost"),
	…
)()
```

Regardless of this rather minor downside, this API allows us to create
subdirectories that act as more specific packages for launching and configuring
specific servers, for instance, an `integration/prosody` package might have an
API like the following:

```
// ConfigFile is an option that can be used to write a temporary Prosody config file.
func ConfigFile(cfg Config) integration.Option

// New creates a new, unstarted, Prosody daemon.
func New(ctx context.Context, opts ...integration.Option) (*integration.Cmd, error)

// Test starts a Prosody instance and returns a function that runs f as a
// subtest using t.Run.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner

// Config contains options that can be written to a Prosody config file.
type Config {
	Admins []string
	VHosts []string
	…
}
```

Internally they would use the `integration` package an its various options.
For example, `prosody.ConfigFile` could be implemented in terms of
`integration.TempFile`.
