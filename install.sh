#!/usr/bin/env bash
# Passenger Go installer.
#
# Installs the binary (downloaded and checksum-verified, or one you give it),
# asks for each setting — checking every answer, and generating the ones you
# would rather not choose — writes the .env and a hardened systemd unit,
# starts the service and waits until it answers.
#
#   sudo ./install.sh                 ask for every setting
#   sudo ./install.sh --yes           take the defaults, generate the secrets
#   ./install.sh --dry-run            show what would happen, change nothing
#   ./install.sh --help               every option
#
# Piped from the web it reads its answers from the terminal:
#   curl -fsSL https://raw.githubusercontent.com/Elagoht/passenger-go/main/install.sh | sudo bash

REPO="Elagoht/passenger-go"
SERVICE_NAME="passenger-go"

INSTALL_DIR="/opt/passenger-go"
SERVICE_USER="passenger"
VERSION="latest"
BINARY=""
ASSUME_YES=false
DRY_RUN=false
SKIP_VERIFY=false
RECONFIGURE=false
PRINT_UNIT=false

# Settings given on the command line; empty means "ask" (or the default with --yes).
OPT_HOST=""
OPT_PORT=""
OPT_CSRF_KEY=""
OPT_SESSION_IDLE=""
OPT_SESSION_MAX=""
OPT_TRUSTED_PROXIES=""
OPT_TRUSTED_PROXIES_SET=false

# ---------------------------------------------------------------- output ----

if [ -t 1 ]; then
    C_BOLD=$'\033[1m' C_DIM=$'\033[2m' C_RED=$'\033[31m' C_GREEN=$'\033[32m'
    C_YELLOW=$'\033[33m' C_BLUE=$'\033[34m' C_RESET=$'\033[0m'
else
    C_BOLD="" C_DIM="" C_RED="" C_GREEN="" C_YELLOW="" C_BLUE="" C_RESET=""
fi

step() { printf '%s==>%s %s%s%s\n' "$C_BLUE" "$C_RESET" "$C_BOLD" "$*" "$C_RESET"; }
info() { printf '    %s\n' "$*"; }
ok()   { printf '    %s✓%s %s\n' "$C_GREEN" "$C_RESET" "$*"; }
warn() { printf '    %s!%s %s\n' "$C_YELLOW" "$C_RESET" "$*" >&2; }
die()  { printf '%serror:%s %s\n' "$C_RED" "$C_RESET" "$*" >&2; exit 1; }
# reason says why an answer was refused.
reason() { printf '    %s✗ %s%s\n' "$C_RED" "$*" "$C_RESET" >&2; }

# run executes a command, or only prints it under --dry-run.
run() {
    if [ "$DRY_RUN" = true ]; then
        printf '    %s+ %s%s\n' "$C_DIM" "$*" "$C_RESET"
        return 0
    fi
    "$@"
}

# ------------------------------------------------------------ validation ----
# Each check_* prints why a value is refused and returns 1, or returns 0.

is_uint() {
    case "$1" in
        '' | *[!0-9]*) return 1 ;;
    esac
    return 0
}

valid_ipv4() {
    local ip="$1" IFS=. part
    [[ "$ip" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] || return 1
    for part in $ip; do
        [ "$((10#$part))" -le 255 ] || return 1
    done
    return 0
}

valid_ipv6() {
    local ip="$1" rest
    [[ "$ip" == *:* ]] || return 1
    [[ "$ip" =~ ^[0-9A-Fa-f:]+$ ]] || return 1
    [ "${#ip}" -le 39 ] || return 1
    # At most one "::".
    rest="${ip#*::}"
    [[ "$rest" != *::* ]] || return 1
    [[ "$ip" != *:::* ]] || return 1
    return 0
}

check_host() {
    case "$1" in
        localhost) return 0 ;;
    esac
    if valid_ipv4 "$1" || valid_ipv6 "$1"; then
        return 0
    fi
    reason "Use an IP address to listen on, such as 127.0.0.1 or 0.0.0.0."
    return 1
}

# port_in_use reports whether something listens on TCP port $1.
port_in_use() {
    if command -v ss >/dev/null 2>&1; then
        [ -n "$(ss -Hltn "sport = :$1" 2>/dev/null)" ]
    elif command -v lsof >/dev/null 2>&1; then
        lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
    else
        return 1
    fi
}

# CURRENT_PORT is the port an existing installation already uses; it may be
# "in use" by this very service.
CURRENT_PORT=""

check_port() {
    if ! is_uint "$1" || [ "${#1}" -gt 5 ] || [ "$((10#$1))" -lt 1 ] || [ "$((10#$1))" -gt 65535 ]; then
        reason "Use a whole number between 1 and 65535."
        return 1
    fi
    if [ "$((10#$1))" != "$CURRENT_PORT" ] && port_in_use "$((10#$1))"; then
        reason "Port $1 is already in use by another program. Pick another, or type \"random\"."
        return 1
    fi
    return 0
}

gen_port() {
    local tries=0 port
    while [ "$tries" -lt 50 ]; do
        port=$(( ( (RANDOM << 15) | RANDOM ) % 40000 + 20000 ))
        if ! port_in_use "$port"; then
            echo "$port"
            return 0
        fi
        tries=$((tries + 1))
    done
    echo "8080"
}

check_csrf_key() {
    local key="$1"
    if [ "${#key}" -lt 32 ]; then
        reason "The key needs at least 32 characters (it has ${#key}). Press Enter to generate one."
        return 1
    fi
    if [ "${#key}" -gt 512 ]; then
        reason "The key is too long; keep it under 512 characters."
        return 1
    fi
    case "$key" in
        *[[:space:]]* | *\"* | *\'* | *\\* | *\$* | *\`* | *\#*)
            reason "Use letters, digits and symbols other than spaces, quotes, \\, \$, \` and #."
            return 1
            ;;
    esac
    return 0
}

gen_csrf_key() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32
    else
        head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
        echo
    fi
}

# duration_seconds converts 1h30m, 15m, 45s... to seconds; it fails on
# anything else (only h, m and s are accepted, which is all a session needs).
duration_seconds() {
    local d="$1" total=0 num unit
    [[ "$d" =~ ^([0-9]+[hms])+$ ]] || return 1
    while [ -n "$d" ]; do
        [[ "$d" =~ ^([0-9]+)([hms])(.*)$ ]] || return 1
        num="${BASH_REMATCH[1]}" unit="${BASH_REMATCH[2]}" d="${BASH_REMATCH[3]}"
        case "$unit" in
            h) total=$((total + 10#$num * 3600)) ;;
            m) total=$((total + 10#$num * 60)) ;;
            s) total=$((total + 10#$num)) ;;
        esac
    done
    echo "$total"
}

check_duration() {
    local secs
    if ! secs=$(duration_seconds "$1") || [ "$secs" -le 0 ]; then
        reason "Write a length of time such as 15m, 1h or 1h30m (h, m and s)."
        return 1
    fi
    return 0
}

# SESSION_IDLE_SECONDS is set once the idle timeout is known, so the maximum
# can be checked against it.
SESSION_IDLE_SECONDS=0

check_session_max() {
    check_duration "$1" || return 1
    local secs
    secs=$(duration_seconds "$1")
    if [ "$secs" -lt "$SESSION_IDLE_SECONDS" ]; then
        reason "The maximum must be at least as long as the idle timeout."
        return 1
    fi
    return 0
}

check_proxy() {
    local addr="$1" ip prefix
    ip="${addr%/*}"
    if [ "$ip" != "$addr" ]; then
        prefix="${addr#*/}"
        is_uint "$prefix" || return 1
        if valid_ipv4 "$ip"; then
            [ "$((10#$prefix))" -le 32 ] || return 1
        elif valid_ipv6 "$ip"; then
            [ "$((10#$prefix))" -le 128 ] || return 1
        else
            return 1
        fi
        return 0
    fi
    valid_ipv4 "$ip" || valid_ipv6 "$ip"
}

check_proxies() {
    local list="$1" item IFS=,
    [ -z "${list// /}" ] && return 0
    for item in $list; do
        item="${item// /}"
        if [ -z "$item" ] || ! check_proxy "$item"; then
            reason "\"$item\" is not an IP address or CIDR range. Separate several with commas, e.g. 127.0.0.1,10.0.0.0/8."
            return 1
        fi
    done
    return 0
}

normalize_proxies() {
    echo "${1// /}"
}

# ---------------------------------------------------------------- prompts ----

# ask LABEL HELP DEFAULT CHECK [GENERATOR]
# Sets ANSWER. Enter takes DEFAULT (or generates one when DEFAULT is empty
# and a generator is given); "random" always generates. Re-asks until CHECK
# accepts the value.
ANSWER=""
ask() {
    local label="$1" help="$2" default="$3" check="$4" gen="${5:-}" hint reply
    [ -n "$help" ] && printf '\n    %s%s%s\n' "$C_DIM" "$help" "$C_RESET"
    if [ -n "$default" ]; then
        hint="[$default]"
        [ -n "$gen" ] && hint="[$default, or \"random\"]"
    elif [ -n "$gen" ]; then
        hint="[Enter to generate]"
    else
        hint="[empty]"
    fi
    while true; do
        printf '    %s %s%s%s: ' "$label" "$C_DIM" "$hint" "$C_RESET"
        if ! IFS= read -r reply </dev/tty; then
            die "No terminal to read answers from. Run with --yes to use the defaults."
        fi
        if [ -z "$reply" ]; then
            if [ -z "$default" ] && [ -n "$gen" ]; then
                reply=$("$gen")
                ok "Generated."
            else
                reply="$default"
            fi
        elif [ -n "$gen" ] && { [ "$reply" = random ] || [ "$reply" = r ]; }; then
            reply=$("$gen")
            ok "Generated: $reply"
        fi
        if "$check" "$reply"; then
            ANSWER="$reply"
            return 0
        fi
    done
}

# settle LABEL GIVEN DEFAULT CHECK [GENERATOR] [HELP]
# A value given on the command line is checked and used; under --yes the
# default (or a generated value) is used; otherwise the reader is asked.
settle() {
    local label="$1" given="$2" default="$3" check="$4" gen="${5:-}" help="${6:-}"
    if [ -n "$given" ]; then
        "$check" "$given" || die "$label: \"$given\" is not valid."
        ANSWER="$given"
    elif [ "$ASSUME_YES" = true ]; then
        if [ -z "$default" ] && [ -n "$gen" ]; then
            ANSWER=$("$gen")
        else
            ANSWER="$default"
        fi
        "$check" "$ANSWER" || die "$label: the default \"$ANSWER\" is not valid here; give one with a flag."
    else
        ask "$label" "$help" "$default" "$check" "$gen"
    fi
}

confirm() {
    [ "$ASSUME_YES" = true ] && return 0
    local reply
    printf '\n    %s %s[Y/n]%s: ' "$1" "$C_DIM" "$C_RESET"
    IFS= read -r reply </dev/tty || return 1
    case "$reply" in
        '' | y | Y | yes | Yes) return 0 ;;
    esac
    return 1
}

# ----------------------------------------------------------------- pieces ----

# env_value FILE KEY prints KEY's value from an env file.
env_value() {
    [ -f "$1" ] || return 0
    sed -n "s/^$2=//p" "$1" | tail -n 1 | sed -e 's/^"\(.*\)"$/\1/' -e "s/^'\(.*\)'\$/\1/"
}

# render_unit prints the systemd unit for INSTALL_DIR, SERVICE_USER and PORT.
render_unit() {
    local dir="$1" user="$2" port="$3" caps
    if is_uint "$port" && [ "$((10#$port))" -lt 1024 ]; then
        caps="# Ports below 1024 need this one capability, and nothing else.
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE"
    else
        caps="CapabilityBoundingSet="
    fi
    cat <<UNIT
[Unit]
Description=Passenger Go, a self-hosted password vault
Documentation=https://github.com/$REPO
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$user
Group=$user
WorkingDirectory=$dir
EnvironmentFile=$dir/.env
ExecStart=$dir/passenger-go
Restart=on-failure
RestartSec=2s
TimeoutStopSec=20s
UMask=0077

# The vault key lives only in this process's memory; keep it there.
LimitCORE=0
$caps
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=$dir/data
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
ProtectClock=true
ProtectHostname=true
ProtectProc=invisible
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
RestrictNamespaces=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

[Install]
WantedBy=multi-user.target
UNIT
}

detect_arch() {
    case "$(uname -m)" in
        x86_64 | amd64) echo amd64 ;;
        aarch64 | arm64) echo arm64 ;;
        *) return 1 ;;
    esac
}

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
}

release_url() {
    if [ "$VERSION" = latest ]; then
        echo "https://github.com/$REPO/releases/latest/download/$1"
    else
        echo "https://github.com/$REPO/releases/download/$VERSION/$1"
    fi
}

# fetch_binary DEST downloads the release binary for this machine and checks
# it against the release's SHA256SUMS.
fetch_binary() {
    local dest="$1" arch asset sums expected actual
    arch=$(detect_arch) || die "This machine's architecture ($(uname -m)) has no release build."
    asset="passenger-go-linux-$arch"
    step "Downloading $asset ($VERSION)"
    if [ "$DRY_RUN" = true ]; then
        run curl -fsSL "$(release_url "$asset")" -o "$dest"
        [ "$SKIP_VERIFY" = true ] || run "verify $asset against $(release_url SHA256SUMS)"
        return 0
    fi
    curl -fsSL "$(release_url "$asset")" -o "$dest" || die "Download failed: $(release_url "$asset")"
    if [ "$SKIP_VERIFY" = true ]; then
        warn "Skipping the checksum check (--skip-verify)."
        return 0
    fi
    sums=$(curl -fsSL "$(release_url SHA256SUMS)") ||
        die "This release has no SHA256SUMS to check the download against. Use a newer release, or --skip-verify if you trust it."
    expected=$(printf '%s\n' "$sums" | awk -v a="$asset" '$2 == a || $2 == "*"a { print $1 }')
    [ -n "$expected" ] || die "SHA256SUMS has no line for $asset."
    actual=$(sha256_of "$dest")
    [ "$actual" = "$expected" ] || die "Checksum mismatch for $asset: expected $expected, got $actual. Not installing it."
    ok "Checksum verified."
}

# wait_healthy HOST PORT polls /healthz until the service answers.
wait_healthy() {
    local host="$1" port="$2" probe
    case "$host" in
        0.0.0.0 | 127.0.0.1 | localhost) probe="127.0.0.1" ;;
        :: | ::1) probe="[::1]" ;;
        *:*) probe="[$host]" ;;
        *) probe="$host" ;;
    esac
    for _ in $(seq 1 30); do
        if curl -fsS --max-time 2 "http://$probe:$port/healthz" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    return 1
}

usage() {
    cat <<USAGE
Install or update Passenger Go as a systemd service.

Usage: install.sh [options]

  -y, --yes                 Do not ask: take the defaults and generate the secrets.
      --dir DIR             Install directory (default $INSTALL_DIR).
      --user NAME           System user the service runs as (default $SERVICE_USER).
      --version TAG         Release to install, e.g. v2.0.0 (default latest).
      --binary FILE         Install this binary instead of downloading one.
      --skip-verify         Do not check the download against SHA256SUMS.
      --reconfigure         Ask for the settings again on an existing install.
      --dry-run             Show what would be done; change nothing.
      --print-unit          Print the systemd unit for these options and exit.

Settings (asked for when not given; checked either way):
      --host ADDR           Address to listen on (127.0.0.1, 0.0.0.0, ::...).
      --port N              Port, 1-65535, or "random" for a free one.
      --csrf-key KEY        Form-signing key, 32+ characters, or "random".
      --session-idle DUR    Sign out after this long without activity (15m).
      --session-max DUR     Sign out this long after signing in (12h).
      --trusted-proxies L   Comma-separated IPs/CIDRs of your reverse proxies.
  -h, --help                This help.
USAGE
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            -y | --yes) ASSUME_YES=true ;;
            --dir) INSTALL_DIR="${2:?--dir needs a value}"; shift ;;
            --user) SERVICE_USER="${2:?--user needs a value}"; shift ;;
            --version) VERSION="${2:?--version needs a value}"; shift ;;
            --binary) BINARY="${2:?--binary needs a value}"; shift ;;
            --skip-verify) SKIP_VERIFY=true ;;
            --reconfigure) RECONFIGURE=true ;;
            --dry-run) DRY_RUN=true ;;
            --print-unit) PRINT_UNIT=true ;;
            --host) OPT_HOST="${2:?--host needs a value}"; shift ;;
            --port) OPT_PORT="${2:?--port needs a value}"; shift ;;
            --csrf-key) OPT_CSRF_KEY="${2:?--csrf-key needs a value}"; shift ;;
            --session-idle) OPT_SESSION_IDLE="${2:?--session-idle needs a value}"; shift ;;
            --session-max) OPT_SESSION_MAX="${2:?--session-max needs a value}"; shift ;;
            --trusted-proxies) OPT_TRUSTED_PROXIES="${2-}"; OPT_TRUSTED_PROXIES_SET=true; shift ;;
            -h | --help) usage; exit 0 ;;
            *) usage >&2; die "Unknown option: $1" ;;
        esac
        shift
    done
    [ "$OPT_PORT" = random ] && OPT_PORT=$(gen_port)
    [ "$OPT_CSRF_KEY" = random ] && OPT_CSRF_KEY=$(gen_csrf_key)
    case "$INSTALL_DIR" in
        /*) ;;
        *) die "--dir must be an absolute path." ;;
    esac
    INSTALL_DIR="${INSTALL_DIR%/}"
    [[ "$SERVICE_USER" =~ ^[a-z_][a-z0-9_-]{0,31}$ ]] || die "--user must be a valid user name."
    if [ -n "$BINARY" ] && [ ! -s "$BINARY" ]; then
        die "--binary $BINARY does not exist or is empty."
    fi
    return 0
}

# ------------------------------------------------------------------- main ----

main() {
    set -euo pipefail
    parse_args "$@"

    if [ "$PRINT_UNIT" = true ]; then
        render_unit "$INSTALL_DIR" "$SERVICE_USER" "${OPT_PORT:-8080}"
        exit 0
    fi

    local env_file="$INSTALL_DIR/.env"
    local old_db="$INSTALL_DIR/database/passenger.db"
    local new_db="$INSTALL_DIR/data/passenger.db"
    local unit_file="/etc/systemd/system/$SERVICE_NAME.service"

    printf '\n%sPassenger Go installer%s\n\n' "$C_BOLD" "$C_RESET"
    [ "$DRY_RUN" = true ] && warn "Dry run: nothing will be changed."

    # -- Checks ---------------------------------------------------------------
    step "Checking this machine"
    if [ "$DRY_RUN" != true ]; then
        [ "$(uname -s)" = Linux ] || die "Passenger Go installs as a systemd service; this is not Linux."
        command -v systemctl >/dev/null 2>&1 || die "systemd (systemctl) is required."
        if [ "$(id -u)" -ne 0 ]; then
            if [ -f "$0" ] && command -v sudo >/dev/null 2>&1; then
                info "Root is needed; running again with sudo."
                exec sudo bash "$0" "$@"
            fi
            die "Run as root, e.g. with sudo."
        fi
        [ -n "$BINARY" ] || command -v curl >/dev/null 2>&1 || die "curl is required to download the release."
        command -v useradd >/dev/null 2>&1 || id "$SERVICE_USER" >/dev/null 2>&1 ||
            die "useradd is required to create the $SERVICE_USER user."
        if [ -z "$BINARY" ]; then
            detect_arch >/dev/null || die "This machine's architecture ($(uname -m)) has no release build."
        fi
    fi
    ok "Ready."

    # An install of the old version keeps its database in database/ and its
    # AES_GCM_SECRET in .env: both are needed to migrate.
    local upgrade=false keep_env=false
    if [ -f "$old_db" ]; then
        upgrade=true
        if [ -e "$INSTALL_DIR/.env.old" ]; then
            die "$INSTALL_DIR/.env.old already exists and may hold your old AES_GCM_SECRET; refusing to overwrite it.
       If an earlier run moved your old .env there, finish the migration steps it printed,
       or move .env.old back to .env by hand, then run this again."
        fi
        [ -f "$env_file" ] || die "Found the old database but not $env_file: its AES_GCM_SECRET is needed to migrate."
        step "Found an installation of the old version"
        info "Its accounts will be migrated after this finishes; nothing of it is deleted."
    elif [ -f "$env_file" ]; then
        if [ "$RECONFIGURE" = true ]; then
            step "Found an existing installation; its settings are the defaults below"
        else
            keep_env=true
            step "Found an existing installation"
            info "Keeping its settings ($env_file). Use --reconfigure to change them."
        fi
    fi

    # -- Settings -------------------------------------------------------------
    local host port csrf_key idle max proxies
    if [ "$keep_env" = true ]; then
        port=$(env_value "$env_file" PORT)
        host=$(env_value "$env_file" HOST)
        port="${port:-8080}"
        host="${host:-localhost}"
    else
        step "Settings"
        local d_host d_port d_idle d_max d_proxies d_key=""
        if [ "$upgrade" = true ]; then
            d_host="0.0.0.0" # the old version listened on every interface
            d_port=$(env_value "$env_file" PORT)
        else
            d_host=$(env_value "$env_file" HOST)
            d_port=$(env_value "$env_file" PORT)
            d_key=$(env_value "$env_file" CSRF_KEY)
        fi
        CURRENT_PORT="$d_port"
        d_host="${d_host:-127.0.0.1}"
        if [ -z "$d_port" ]; then
            d_port=8080
            port_in_use 8080 && d_port=$(gen_port)
        fi
        d_idle=$(env_value "$env_file" SESSION_IDLE)
        d_max=$(env_value "$env_file" SESSION_MAX)
        d_proxies=$(env_value "$env_file" TRUSTED_PROXIES)

        settle "Listen address" "$OPT_HOST" "$d_host" check_host "" \
            "127.0.0.1 keeps it on this machine, for a reverse proxy in front (recommended); 0.0.0.0 opens it to every network."
        host="$ANSWER"

        settle "Port" "$OPT_PORT" "$d_port" check_port gen_port \
            "The port the service listens on. Type \"random\" for a free one."
        port="$((10#$ANSWER))"

        settle "Form key (CSRF_KEY)" "$OPT_CSRF_KEY" "$d_key" check_csrf_key gen_csrf_key \
            "Signs the forms. Nobody needs to know it: press Enter to generate a random one."
        csrf_key="$ANSWER"

        settle "Sign out when idle for" "$OPT_SESSION_IDLE" "${d_idle:-15m}" check_duration "" \
            "How long the vault stays unlocked without activity, e.g. 15m, 1h."
        idle="$ANSWER"
        SESSION_IDLE_SECONDS=$(duration_seconds "$idle")

        settle "Sign out at most after" "$OPT_SESSION_MAX" "${d_max:-12h}" check_session_max "" \
            "How long a sign-in lasts at most, whatever the activity, e.g. 12h."
        max="$ANSWER"

        if [ -z "$d_proxies" ]; then
            case "$host" in
                127.0.0.1 | localhost | ::1) d_proxies="127.0.0.1" ;;
            esac
        fi
        if [ "$OPT_TRUSTED_PROXIES_SET" = true ]; then
            check_proxies "$OPT_TRUSTED_PROXIES" || die "--trusted-proxies: not a list of IPs or CIDR ranges."
            proxies="$OPT_TRUSTED_PROXIES"
        else
            settle "Trusted proxies" "" "$d_proxies" check_proxies "" \
                "Reverse proxies whose X-Forwarded-For names the real client. Leave empty when nothing is in front."
            proxies="$ANSWER"
        fi
        proxies=$(normalize_proxies "$proxies")

        printf '\n    %sSummary%s\n' "$C_BOLD" "$C_RESET"
        info "Install directory  $INSTALL_DIR (runs as user \"$SERVICE_USER\")"
        info "Listen on          $host:$port"
        info "Form key           ${csrf_key:0:6}… (${#csrf_key} characters)"
        info "Sessions           idle $idle, at most $max"
        info "Trusted proxies    ${proxies:-none}"
        if [ "$host" = 0.0.0.0 ] || [ "$host" = "::" ]; then
            warn "Every network can reach it over plain HTTP. Put it behind HTTPS (a reverse proxy) before using it from elsewhere."
        fi
        confirm "Install with these settings?" || die "Stopped; nothing was changed."
    fi

    # -- User and directories -------------------------------------------------
    step "Preparing $INSTALL_DIR"
    if ! id "$SERVICE_USER" >/dev/null 2>&1; then
        run useradd --system --home-dir "$INSTALL_DIR" --no-create-home \
            --shell /usr/sbin/nologin --user-group "$SERVICE_USER"
        ok "Created system user $SERVICE_USER."
    fi
    run install -d -m 755 -o root -g root "$INSTALL_DIR"
    run install -d -m 700 -o "$SERVICE_USER" -g "$SERVICE_USER" "$INSTALL_DIR/data"

    # -- Binary -----------------------------------------------------------------
    local tmp_bin="$INSTALL_DIR/passenger-go.new"
    if [ -n "$BINARY" ]; then
        step "Using $BINARY"
        run install -m 755 -o root -g root "$BINARY" "$tmp_bin"
    else
        fetch_binary "$tmp_bin"
        run chown root:root "$tmp_bin"
        run chmod 755 "$tmp_bin"
    fi

    if [ "$upgrade" = true ]; then
        # Only now that the new binary is here: stop the old service and keep
        # it from starting at boot, so it cannot create data/passenger.db
        # before migrate runs.
        run systemctl disable --now "$SERVICE_NAME" || true
    fi
    run mv -f "$tmp_bin" "$INSTALL_DIR/passenger-go"
    ok "Installed $INSTALL_DIR/passenger-go"

    # -- Configuration ------------------------------------------------------------
    if [ "$upgrade" = true ]; then
        run mv "$env_file" "$INSTALL_DIR/.env.old"
        run chown "root:$SERVICE_USER" "$INSTALL_DIR/.env.old"
        run chmod 640 "$INSTALL_DIR/.env.old"
        run chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/database"
    fi
    if [ "$keep_env" != true ]; then
        step "Writing $env_file"
        local content
        content="# Written by install.sh. Run it with --reconfigure to change these.
HOST=$host
PORT=$port
DATABASE_PATH=$new_db
CSRF_KEY=$csrf_key
SESSION_IDLE=$idle
SESSION_MAX=$max
TRUSTED_PROXIES=$proxies"
        if [ "$DRY_RUN" = true ]; then
            printf '%s\n' "$content" | sed -e "s/^CSRF_KEY=.*/CSRF_KEY=<${#csrf_key} characters>/" -e 's/^/      /'
        else
            (
                umask 077
                printf '%s\n' "$content" >"$env_file.new"
            )
            chown "root:$SERVICE_USER" "$env_file.new"
            chmod 640 "$env_file.new"
            mv -f "$env_file.new" "$env_file"
        fi
        ok "Readable by root and $SERVICE_USER only."
    fi

    # -- systemd -----------------------------------------------------------------
    step "Installing the systemd unit"
    if [ "$DRY_RUN" = true ]; then
        info "$unit_file:"
        render_unit "$INSTALL_DIR" "$SERVICE_USER" "$port" | sed 's/^/      /'
    else
        render_unit "$INSTALL_DIR" "$SERVICE_USER" "$port" >"$unit_file.new"
        chmod 644 "$unit_file.new"
        mv -f "$unit_file.new" "$unit_file"
        if command -v systemd-analyze >/dev/null 2>&1; then
            systemd-analyze verify "$unit_file" >/dev/null 2>&1 ||
                warn "systemd-analyze reports a problem with the unit: systemd-analyze verify $unit_file"
        fi
    fi
    run systemctl daemon-reload
    ok "$unit_file"

    if [ "$upgrade" = true ]; then
        printf '\n%sAlmost done: migrate your accounts.%s The service is stopped and will not start until you do.\n\n' "$C_YELLOW" "$C_RESET"
        info "1. Migrate (your master passphrase stays the same):"
        info "     sudo -u $SERVICE_USER $INSTALL_DIR/passenger-go migrate --from $old_db --env $INSTALL_DIR/.env.old --to $new_db"
        info "2. Write down the NEW recovery key it prints; the old one stops working."
        info "3. Delete the old files; they hold the old secret:"
        info "     sudo rm -rf $INSTALL_DIR/database $INSTALL_DIR/.env.old"
        info "4. Start the service:"
        info "     sudo systemctl enable --now $SERVICE_NAME"
        echo
        exit 0
    fi

    # -- Start and check ----------------------------------------------------------
    step "Starting $SERVICE_NAME"
    if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        run systemctl restart "$SERVICE_NAME"
    else
        run systemctl enable --now "$SERVICE_NAME"
    fi
    if [ "$DRY_RUN" = true ]; then
        run curl -fsS "http://$host:$port/healthz"
        printf '\n%sDry run finished.%s\n' "$C_GREEN" "$C_RESET"
        exit 0
    fi
    if ! wait_healthy "$host" "$port"; then
        journalctl -u "$SERVICE_NAME" -n 30 --no-pager >&2 || true
        die "The service did not answer on $host:$port/healthz within 30 seconds; its log is above."
    fi
    ok "Answering on http://$host:$port"

    local code
    code=$(journalctl -u "$SERVICE_NAME" --no-pager -o cat 2>/dev/null | sed -n 's/.*setup code: \([0-9]\{8\}\).*/\1/p' | tail -n 1 || true)
    printf '\n%sPassenger Go is running.%s\n\n' "$C_GREEN" "$C_RESET"
    if [ -n "$code" ]; then
        info "Open http://$host:$port and create your vault with this setup code:"
        printf '\n        %s%s%s\n\n' "$C_BOLD" "$code" "$C_RESET"
        info "A new code is printed on every start until the vault exists; the newest is:"
        info "  journalctl -u $SERVICE_NAME | grep 'setup code' | tail -n 1"
    fi
    info "Logs:     journalctl -u $SERVICE_NAME -f"
    info "Settings: $env_file (run this script with --reconfigure to change them)"
    echo
}

# Sourced by the tests for its functions; run as a program otherwise.
if [ "${PASSENGER_INSTALL_LIB:-}" = 1 ]; then
    # shellcheck disable=SC2317 # exit is reached only when run, not sourced
    return 0 2>/dev/null || exit 0
fi
main "$@"
