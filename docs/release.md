# Release Notes

When making a release, remember to do the following:

- Create a new commit bumping the release in `CHANGELOG.md` with the commit
  message "all: release vX.Y.Z"
- Tag the release with `git tag -s -a --cleanup=whitespace vX.Y.Z` and remove
  all comments from the annotation as these won't be treated as comments and
  will end up in the final annotation
- Copy the changelog for the release in as the annotation, making sure the
  headers are correct (ie. remove one `#' from each header)
- Create a new release on Codeberg (https://codeberg.org/mellium/xmpp/releases)
  using the same annotation, making sure headers make sense (remove the top
  level one, Codeberg will create that from the tag)
- Do a `go get mellium.im/xmpp@release` (on a machine that has not disabled the
  proxy) to trigger the docs being built
- Write up a release announcement on https://opencollective.com/mellium
- Announce the release
  - Post it in `users@mellium.chat`
  - Post it on fedi at https://social.coop/@mellium
  - Post it on Reddit https://www.reddit.com/r/xmpp/
  - If there's anything worth demoing sign up for Office Hours
    https://wiki.xmpp.org/web/XMPP_Office_Hours
