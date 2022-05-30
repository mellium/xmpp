# Security Policy

## Supported Versions

Only the latest major version is supported.
Older versions will not receive security fixes.


## Reporting a Vulnerability

Security sensitive issues should be reported directly to the project maintainer
by emailing [`security@mellium.im`].
A maintainer will respond to your report within 48 hours.

## Verifying Releases

All releases will be tagged and signed with one of the following GPG signing
keys:

```
82214D7FB54DC9A3BC0CDAE116D5138E52B849B3
```

Keys may be pulled from your keyserver of choice and verifications can be
performed using Git:

```
$ gpg --recv-keys 82214D7FB54DC9A3BC0CDAE116D5138E52B849B3
$ git verify-tag v0.21.4
```

[`security@mellium.im`]: mailto:security@mellium.im
