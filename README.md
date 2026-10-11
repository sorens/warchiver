# warchiver

A command-line tool for archiving World of Warcraft™ account and character data.

## Quick Start

### 1. Authenticate with Battle.net

Create an application at [develop.battle.net](https://develop.battle.net/) and note your client ID and secret. Then run:

```bash
warchiver auth login -client-id <ID> -client-secret <SECRET>
```

This opens your browser for Battle.net authorization and stores your token locally in your keystore.

### 2. Archive

```bash
# Archive a single character (region defaults to us)
warchiver archive character <realm> <character>

# Same, with explicit region
warchiver archive character -region us <realm> <character>

# Archive every character on every WoW account attached to your Battle.net account
warchiver archive account
```

Archives are written as `warchive.<timestamp>.<hash>.tar.gz` to `archive_dir` in `config.json`. If `archive_dir` is unset, they go in the `archives` subdirectory of the platform config directory listed below.

## Commands

- `auth`
- `account`
- `archive` 

Flags are specific to each subcommand. Use `warchiver <command> <subcommand> -help` to see the available flags.

### `auth` — Battle.net authentication

- `auth info` — Show the authenticated account's info.
- `auth login` — Start browser-based OAuth authorization.
  - Flags: 
    - `-client-id`
    - `-client-secret`
  - ENV: 
    - `WARCHIVER_BNET_CLIENT_ID`
    - `WARCHIVER_BNET_CLIENT_SECRET`
- `auth logout` (alias `clear`) — Remove the stored access token.

OAuth scopes: `wow.profile openid`.

### `account` — account inspection

- `account list` (alias `ls`) — List characters across all WoW accounts on the Battle.net account.
  - Flags:
    - `-config`
    - `-region`
    - `-verbose`
    - `-minlvl <n>` (default `1`)

### `archive` — produce a warchive bundle

- `archive account` (aliases `acct`, `a`) — Archive eligible characters across all realms and account-wide data.
  - Flags: 
    - `-config`
    - `-region`
    - `-verbose`
    - `-game`
    - `-minlvl <n>`

- `archive character <realm> <character>` (aliases `char`, `c`) — Archive one character.
  - Flags: 
    - `-config`
    - `-region`
    - `-verbose`
    - `-game`


Only `archive account` accepts `-minlvl`.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `WARCHIVER_BNET_CLIENT_ID` | Battle.net client ID for `auth login` |
| `WARCHIVER_BNET_CLIENT_SECRET` | Battle.net client secret for `auth login` |
| `WARCHIVER_DIR` | Base directory for temporary working files (debugging) |
| `WARCHIVER_TOKEN` | Inject an access token; bypasses OAuth |
| `WARCHIVER_TOKEN_EXPIRY` | Expiry timestamp paired with `WARCHIVER_TOKEN` |

Config file location (per platform):

- macOS: `~/Library/Application Support/com.warchiver/`
- Windows: `%APPDATA%\com.warchiver\`
- Linux: `~/.config/com.warchiver/`

Battle.net client credentials and OAuth tokens are stored in the platform credential store (macOS Keychain, Windows Credential Manager, or the Linux Secret Service). Other settings remain in `config.json`.

## Requirements

- **Go 1.26+** (to build from source; see `go.mod`)
- A [Battle.net Developer](https://develop.battle.net/) application with client ID and secret
