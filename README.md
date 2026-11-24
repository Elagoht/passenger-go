<div align="center">
  <img src="https://raw.githubusercontent.com/Elagoht/passenger-go/refs/heads/main/static/passenger.png" alt="Passenger-Go Logo" width="128" height="128">

# Passenger-Go

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white)
![HTML](https://img.shields.io/badge/HTML-E34F26?style=for-the-badge&logo=html5&logoColor=white)
![CSS](https://img.shields.io/badge/CSS-1572B6?style=for-the-badge&logo=css&logoColor=white)
![JavaScript](https://img.shields.io/badge/JavaScript-F7DF1E?style=for-the-badge&logo=javascript&logoColor=black)
![Tailscale](https://img.shields.io/badge/Tailscale-000000?style=for-the-badge&logo=tailscale&logoColor=white)
---

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/Elagoht/passenger-go/release.yaml?style=for-the-badge)
[![GitHub Stars](https://img.shields.io/github/stars/Elagoht/passenger-go.svg?style=for-the-badge)](https://github.com/Elagoht/passenger-go/stargazers)
![GitHub License](https://img.shields.io/github/license/Elagoht/passenger-go?style=for-the-badge)
</div>

<div align="center">

[![](https://img.shields.io/badge/Download_CLI_App-gold?style=for-the-badge)](https://github.com/Elagoht/passenger-go-cli)

</div>

Passenger-Go is a self-hosted, single-user passphrase manager. It is one Go binary with the templates and static files embedded, a server-rendered web interface and a SQLite database. There is no separate backend, no JSON API and nothing to copy next to the binary.

## Key Features

- Argon2id-derived vault key; every account is encrypted with AES-GCM
- Master passphrase and one-time recovery key
- Idle and absolute session expiry with a clear "session expired" experience
- CSRF protection, strict security headers, brute-force back-off (including setup)
- Bans for clients that probe for files or keep guessing, and `/healthz` and `/readyz` for health checks
- SQLite database, a single file to back up
- Single static binary, no CGO
- Responsive UI with dark/light mode, real-time search, favicons on account cards
- Passphrase generator, strength meter and look-alike character alternator
- Import/export of Firefox and Chromium CSV files
- `migrate` command for moving a vault from the previous (JWT / `AES_GCM_SECRET`) version

## Security Model

The vault key is derived from your master passphrase with Argon2id and is never stored. A copy of the database plus the server configuration (`.env`) is not enough to decrypt anything: without the master passphrase (or the recovery key) the accounts stay encrypted. The unlocked key lives only in server memory for the duration of a session, so restarting the server signs you out and you unlock the vault again with your passphrase.

Over plain HTTP the session cookie cannot be marked `Secure`. For anything beyond a private network such as Tailscale, run Passenger-Go behind a TLS reverse proxy with `HOST=127.0.0.1` and `TRUSTED_PROXIES` set to the proxy's address.

Wrong passphrase, recovery-key and setup-code guesses are throttled per client address (an IPv6 client counts as its whole /64). Behind a proxy every request comes from the proxy's address, so unless `TRUSTED_PROXIES` names the proxy, all clients share one budget and a few wrong guesses from anyone make everyone wait. A client that keeps guessing past the back-off, 100 wrong guesses within a day with no right one between them (about seven hours of guessing as fast as the back-off allows), is banned from the whole site for a day. Requests from loopback (`127.0.0.0/8`, `::1`) are never banned, so a proxy on the same machine cannot get every client banned, but the back-off still applies to them.

## Recommended Way to Run

Passenger-Go is designed to run as an always-online server on a VPS or a dedicated machine. [Tailscale](https://tailscale.com) is a good way to reach it from anywhere without exposing it to the internet.

## Install (Linux, systemd)

```bash
curl -fsSL https://raw.githubusercontent.com/Elagoht/passenger-go/main/install.sh | sudo bash
```

or from a clone: `sudo ./install.sh`. The installer:

- downloads the release binary for your machine and checks it against the release's `SHA256SUMS` (or installs one you give it with `--binary`);
- asks for each setting and checks every answer, re-asking with the reason when one is wrong. You do not need to know them: Enter takes the default, and the secrets are generated for you:

  | Setting | Default | |
  |---|---|---|
  | Listen address | `127.0.0.1` | `0.0.0.0` opens it to every network |
  | Port | `8080`, or a free one | type `random` for a free port; a port already in use is refused |
  | Form key (`CSRF_KEY`) | generated | press Enter; your own must be 32+ characters |
  | Idle sign-out | `15m` | `h`, `m`, `s`, e.g. `1h30m` |
  | Maximum session | `12h` | at least the idle time |
  | Trusted proxies | `127.0.0.1` when listening locally | IPs or CIDR ranges, comma-separated |

- runs the service as its own system user, `passenger`, under a hardened systemd unit ([`passenger-go.service`](passenger-go.service)): the binary is root-owned and read-only, the service can write only to `/opt/passenger-go/data` (mode 700), and `.env` is readable by root and the service only (mode 640);
- starts it, waits until it answers on `/healthz`, and prints the setup code.

Run it again to update the binary: your settings are kept. `--reconfigure` asks for them again (the current values are the defaults), `--yes` takes the defaults without asking, `--dry-run` shows everything it would do without changing anything, and `--help` lists every option, including a flag for each setting.

Then serve it on your Tailscale network:

```bash
tailscale serve --bg [YOUR_PORT_HERE]
```

Whatever proxies the requests to Passenger-Go (a reverse proxy, or `tailscale serve`) must send an `X-Forwarded-For` header, and `TRUSTED_PROXIES` must name the proxy's address, `127.0.0.1` when it runs on the same machine (the installer's default when listening locally). Otherwise every client shares the proxy's address and with it one rate-limit budget for passphrase guesses. A proxy on the same machine is never banned (loopback addresses are exempt), so one client probing for files cannot get everyone banned, but it then goes unbanned too; a proxy on another machine that is not named gets banned, and with it everyone.

### First run: setup code

Until the vault is set up, the server logs a setup code every time it starts (a new one on each start; only the newest works). Setting up the vault (choosing the master passphrase) requires it, so nobody else can claim a fresh server before you. Use the newest line:

```bash
journalctl -u passenger-go | grep "setup code" | tail -n 1
```

The installer prints it when it finishes. Open the site, enter the code, choose your master passphrase and store the recovery key it shows you somewhere safe.

## Upgrading from the previous version

The old version stored everything under one `AES_GCM_SECRET` in `.env`. Running the installer on such a machine (it finds `/opt/passenger-go/database/passenger.db`) keeps listening on every interface and on the old port by default, and:

1. Once the new binary is downloaded and verified, stops and disables the old service, so neither a restart nor a reboot starts the new binary before the migration.
2. Moves the old `.env` to `/opt/passenger-go/.env.old` (it refuses to run if `.env.old` already exists), hands the old files to the `passenger` user and writes a fresh `.env`.
3. Stops without starting the service, and prints the migration steps, which are:

```bash
sudo -u passenger /opt/passenger-go/passenger-go migrate \
  --from /opt/passenger-go/database/passenger.db \
  --env /opt/passenger-go/.env.old \
  --to /opt/passenger-go/data/passenger.db
```

4. Your master passphrase stays the same. `migrate` prints a **new recovery key**; the old one no longer works, so store the new one.
5. After a successful migration delete the old files, since they hold the old secret, and start the service:

```bash
sudo rm -rf /opt/passenger-go/database /opt/passenger-go/.env.old
sudo systemctl enable --now passenger-go
```

`migrate` only reads the old files, and refuses to overwrite an existing target database. It writes the new database under a temporary name (`passenger.db.tmp-…`) and moves it into place only once it is complete and verified, so the target never holds a half-written vault: after Ctrl-C or a failure it cleans up, and you just run it again. If the process was killed outright (or the machine lost power), delete any leftover `/opt/passenger-go/data/passenger.db.tmp-*` files before running it again.

## Environment Variables

Set in `/opt/passenger-go/.env` (see `.env.example`); the installer writes them, and `sudo ./install.sh --reconfigure` changes them. `JWT_SECRET`, `AES_GCM_SECRET` and `SALT` are no longer used.

| Variable | Default | Description |
|---|---|---|
| `CSRF_KEY` | none, required | At least 32 bytes, e.g. `openssl rand -hex 32` |
| `PORT` | `8080` | Port to listen on |
| `HOST` | `localhost` | Address to listen on. The installer asks; `127.0.0.1` by default, `0.0.0.0` when upgrading from the old version (which listened on all interfaces) |
| `DATABASE_PATH` | `./data/passenger.db` | SQLite file; the directory is created (0700), the file is 0600 |
| `SESSION_IDLE` | `15m` | Sign out after this much inactivity |
| `SESSION_MAX` | `12h` | Absolute session lifetime |
| `TRUSTED_PROXIES` | empty | Comma-separated proxy IPs/CIDRs whose `X-Forwarded-For` is trusted |

## Development

Requires Go 1.26.11 or newer.

```bash
cp .env.example .env   # set CSRF_KEY
collage dev            # reads templates and static files from disk
# or
go run .               # uses the embedded files
go test ./... && go vet ./...
```

## License

This project is licensed under the [GPL-3.0](LICENSE) license.

## Open Source

This project is open source and free to use. Feel free to contribute to the project.

## Why Not Saying "Password"

Passwords are obsolete and not secure. Passphrases are more secure and easier to remember. word means, a single word. phrase means, a group of words. How long is your "passkey" means how secure it is. By using phrases, you can remember them easier while being more secure.
