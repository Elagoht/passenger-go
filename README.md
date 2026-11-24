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
- SQLite database, a single file to back up
- Single static binary, no CGO
- Responsive UI with dark/light mode, real-time search, favicons on account cards
- Passphrase generator, strength meter and look-alike character alternator
- Import/export of Firefox and Chromium CSV files
- `migrate` command for moving a vault from the previous (JWT / `AES_GCM_SECRET`) version

## Security Model

The vault key is derived from your master passphrase with Argon2id and is never stored. A copy of the database plus the server configuration (`.env`) is not enough to decrypt anything: without the master passphrase (or the recovery key) the accounts stay encrypted. The unlocked key lives only in server memory for the duration of a session, so restarting the server signs you out and you unlock the vault again with your passphrase.

Over plain HTTP the session cookie cannot be marked `Secure`. For anything beyond a private network such as Tailscale, run Passenger-Go behind a TLS reverse proxy with `HOST=127.0.0.1` and `TRUSTED_PROXIES` set to the proxy's address.

Wrong passphrase, recovery-key and setup-code guesses are throttled per client address (an IPv6 client counts as its whole /64). Behind a proxy every request comes from the proxy's address, so unless `TRUSTED_PROXIES` names the proxy, all clients share one budget and a few wrong guesses from anyone make everyone wait.

## Recommended Way to Run

Passenger-Go is designed to run as an always-online server on a VPS or a dedicated machine. [Tailscale](https://tailscale.com) is a good way to reach it from anywhere without exposing it to the internet.

## Install (Linux, systemd)

```bash
git clone https://github.com/Elagoht/passenger-go.git
cd passenger-go
./install.sh
```

The script downloads the latest release binary to `/opt/passenger-go`, installs the systemd unit, asks for the port (default 8080), generates a random `CSRF_KEY` itself and writes `/opt/passenger-go/.env` (mode 600). The database lives in `/opt/passenger-go/data` (mode 700). It needs `sudo`, `curl` and `openssl`.

Then serve it on your Tailscale network:

```bash
tailscale serve --bg [YOUR_PORT_HERE]
```

If whatever proxies the requests to Passenger-Go (a reverse proxy, or `tailscale serve`) sends an `X-Forwarded-For` header, set `TRUSTED_PROXIES` in `/opt/passenger-go/.env` to the proxy's address, `127.0.0.1` when it runs on the same machine, and restart the service. Otherwise every client shares the proxy's address and with it one rate-limit budget for passphrase guesses.

### First run: setup code

Until the vault is set up, the server logs a setup code every time it starts (a new one on each start; only the newest works). Setting up the vault (choosing the master passphrase) requires it, so nobody else can claim a fresh server before you. Use the newest line:

```bash
journalctl -u passenger-go | grep "setup code" | tail -n 1
```

Open the site, enter the code, choose your master passphrase and store the recovery key it shows you somewhere safe.

## Upgrading from the previous version

The old version stored everything under one `AES_GCM_SECRET` in `.env`. Running `install.sh` on such a machine (it finds `/opt/passenger-go/database/passenger.db`) does the following and then stops without starting the service:

1. Once the new binary is downloaded, stops and disables the old service, so neither a restart nor a reboot starts the new binary before the migration.
2. Moves the old `.env` to `/opt/passenger-go/.env.old` (it refuses to run if `.env.old` already exists) and writes a fresh `.env`.
3. Prints the migration steps, which are:

```bash
/opt/passenger-go/passenger-go migrate \
  --from /opt/passenger-go/database/passenger.db \
  --env /opt/passenger-go/.env.old \
  --to /opt/passenger-go/data/passenger.db
```

4. Your master passphrase stays the same. `migrate` prints a **new recovery key**; the old one no longer works, so store the new one.
5. After a successful migration delete the old files, since they hold the old secret:

```bash
rm -rf /opt/passenger-go/database /opt/passenger-go/.env.old
sudo systemctl enable --now passenger-go.service
```

`migrate` only reads the old files, and refuses to overwrite an existing target database. It writes the new database under a temporary name and moves it into place only once it is complete and verified, so an interrupted run (Ctrl-C included) leaves no half-written vault behind; just run it again.

## Environment Variables

Set in `/opt/passenger-go/.env` (see `.env.example`). `JWT_SECRET`, `AES_GCM_SECRET` and `SALT` are no longer used.

| Variable | Default | Description |
|---|---|---|
| `CSRF_KEY` | none, required | At least 32 bytes, e.g. `openssl rand -hex 32` |
| `PORT` | `8080` | Port to listen on |
| `HOST` | `localhost` | Address to listen on. `install.sh` sets `0.0.0.0` (the old version listened on all interfaces) |
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
