# hidemail

A tiny CLI for creating and managing [iCloud Hide My Email](https://support.apple.com/en-us/105078) addresses from your terminal.

```
$ hidemail gen --label "newsletter"
random.alias@icloud.com
```

Requires an **iCloud+** account (any paid iCloud plan). Built with the Go standard library — no dependencies.

## Install

```
go install .        # from a clone, or:
go build -o hidemail .
```

## Usage

```
hidemail auth                       Connect to iCloud (one time)
hidemail gen [--label L] [--note N] Generate a new address
hidemail list [--active] [--json]   List your addresses
hidemail logout                     Delete the stored session
```

Run `hidemail --help` for all flags.

## Authenticating

Apple has no public API for Hide My Email, so `hidemail` reuses your iCloud **web session**.
Running `hidemail auth` starts a small local page and opens it in your browser: sign in at
[icloud.com](https://www.icloud.com), copy the `Cookie` header from DevTools → Network, and
paste it back. The page posts it to the CLI over `localhost` — nothing leaves your machine.

The session is saved to `~/.config/hidemail/cookies.txt` (owner-only, `0600`) and typically
stays valid for about two weeks. When a command reports the session expired, run
`hidemail auth` again.

## Notes

- Unofficial: this talks to Apple's private iCloud endpoints, which can change without notice.
- The saved session is a bearer credential for your iCloud web session (not your password, and
  it can't change your password or bypass 2FA). Use `hidemail logout` to remove it.
