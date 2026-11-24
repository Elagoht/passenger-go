#!/usr/bin/env bash
# Tests for install.sh: its checks and generators, and whole runs under
# --dry-run (which changes nothing, so it runs anywhere, macOS included).
#
#   bash scripts/install_test.sh

cd "$(dirname "$0")/.." || exit 1
INSTALLER="$PWD/install.sh"

fails=0
pass() { printf '  ok    %s\n' "$1"; }
fail() { printf '  FAIL  %s\n' "$1"; fails=$((fails + 1)); }

expect_ok()  { if "$@" 2>/dev/null; then pass "$*"; else fail "$* (want accepted)"; fi; }
expect_bad() { if "$@" 2>/dev/null; then fail "$* (want refused)"; else pass "refuses: $*"; fi; }

contains() { # name haystack needle
    case "$2" in
        *"$3"*) pass "$1" ;;
        *) fail "$1: missing \"$3\""; printf '%s\n' "$2" | sed 's/^/        | /' ;;
    esac
}
lacks() {
    case "$2" in
        *"$3"*) fail "$1: unexpected \"$3\"" ;;
        *) pass "$1" ;;
    esac
}

# ---------------------------------------------------------- functions ----
(
    PASSENGER_INSTALL_LIB=1
    # shellcheck source=../install.sh
    . "$INSTALLER"
    port_in_use() { [ "$1" = 4444 ]; } # pretend 4444 is taken

    echo "checks"
    expect_ok check_host 127.0.0.1
    expect_ok check_host 0.0.0.0
    expect_ok check_host localhost
    expect_ok check_host ::
    expect_ok check_host ::1
    expect_ok check_host fe80::1
    expect_bad check_host 256.1.1.1
    expect_bad check_host 1.2.3
    expect_bad check_host example.com
    expect_bad check_host 1:::2
    expect_bad check_host ""

    expect_ok check_port 8080
    expect_ok check_port 1
    expect_ok check_port 65535
    expect_bad check_port 0
    expect_bad check_port 65536
    expect_bad check_port 80a
    expect_bad check_port ""
    expect_bad check_port 4444
    CURRENT_PORT=4444
    expect_ok check_port 4444 # the existing install's own port

    expect_ok check_csrf_key "0123456789abcdef0123456789abcdef"
    expect_bad check_csrf_key "short"
    expect_bad check_csrf_key "0123456789abcdef0123456789abcdef with space"
    expect_bad check_csrf_key '0123456789abcdef0123456789abcdef$x'
    expect_bad check_csrf_key '0123456789abcdef0123456789abcdef"'
    expect_bad check_csrf_key '0123456789abcdef0123456789abcdef#'

    expect_ok check_duration 15m
    expect_ok check_duration 1h30m
    expect_ok check_duration 45s
    expect_bad check_duration 0m
    expect_bad check_duration 15
    expect_bad check_duration 1.5h
    expect_bad check_duration 15min
    expect_bad check_duration ""
    [ "$(duration_seconds 1h30m)" = 5400 ] && pass "1h30m is 5400s" || fail "1h30m is 5400s"
    [ "$(duration_seconds 2h15s)" = 7215 ] && pass "2h15s is 7215s" || fail "2h15s is 7215s"

    SESSION_IDLE_SECONDS=3600
    expect_ok check_session_max 1h
    expect_ok check_session_max 12h
    expect_bad check_session_max 30m

    expect_ok check_proxies ""
    expect_ok check_proxies 127.0.0.1
    expect_ok check_proxies "127.0.0.1, 10.0.0.0/8"
    expect_ok check_proxies "::1,fd00::/8"
    expect_bad check_proxies 10.0.0.0/33
    expect_bad check_proxies "127.0.0.1,,10.0.0.1"
    expect_bad check_proxies proxy.local
    [ "$(normalize_proxies "127.0.0.1, 10.0.0.0/8")" = "127.0.0.1,10.0.0.0/8" ] &&
        pass "proxies normalised" || fail "proxies normalised"

    echo "generators"
    key=$(gen_csrf_key)
    check_csrf_key "$key" && [ "${#key}" = 64 ] && pass "generated key is 64 valid characters" ||
        fail "generated key: $key"
    [ "$key" != "$(gen_csrf_key)" ] && pass "keys differ" || fail "keys differ"
    p=$(gen_port)
    check_port "$p" && [ "$p" -ge 20000 ] && [ "$p" -lt 60000 ] && pass "random port $p is free and in range" ||
        fail "random port $p"

    exit "$fails"
) || fails=$((fails + $?))

# ---------------------------------------------------------- the unit -----
echo "unit"
unit=$(bash "$INSTALLER" --print-unit)
if [ "$unit" = "$(cat passenger-go.service)" ]; then
    pass "passenger-go.service matches the unit install.sh writes"
else
    fail "passenger-go.service drifted from install.sh: regenerate it with ./install.sh --print-unit > passenger-go.service"
fi
contains "no capabilities by default" "$unit" "CapabilityBoundingSet="
lacks "no bind capability on 8080" "$unit" "CAP_NET_BIND_SERVICE"
contains "writes only to data" "$unit" "ReadWritePaths=/opt/passenger-go/data"
contains "runs as passenger" "$unit" "User=passenger"
low=$(bash "$INSTALLER" --print-unit --port 443)
contains "port 443 gets the bind capability" "$low" "AmbientCapabilities=CAP_NET_BIND_SERVICE"
custom=$(bash "$INSTALLER" --print-unit --dir /srv/pg/ --user vault)
contains "custom dir" "$custom" "ExecStart=/srv/pg/passenger-go"
contains "custom user" "$custom" "User=vault"

# ---------------------------------------------------------- dry runs -----
echo "dry runs"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
fake_bin="$tmp/passenger-go-bin"
printf 'fake' >"$fake_bin"

out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/fresh" --binary "$fake_bin" --port 18080 2>&1)
status=$?
[ "$status" = 0 ] && pass "fresh --yes dry run succeeds" || fail "fresh --yes dry run exit $status"
contains "listens on loopback by default" "$out" "HOST=127.0.0.1"
contains "keeps the given port" "$out" "PORT=18080"
contains "generates a 64-character key" "$out" "CSRF_KEY=<64 characters>"
contains "default idle" "$out" "SESSION_IDLE=15m"
contains "default max" "$out" "SESSION_MAX=12h"
contains "trusts the local proxy" "$out" "TRUSTED_PROXIES=127.0.0.1"
contains "database under data" "$out" "DATABASE_PATH=$tmp/fresh/data/passenger.db"
contains "prints the unit" "$out" "ExecStart=$tmp/fresh/passenger-go"
contains "starts the service" "$out" "systemctl enable --now passenger-go"
[ ! -e "$tmp/fresh" ] && pass "dry run created nothing" || fail "dry run created $tmp/fresh"

out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/open" --binary "$fake_bin" --port 18081 --host 0.0.0.0 --trusted-proxies "" 2>&1)
contains "open host is written" "$out" "HOST=0.0.0.0"
contains "open host warns" "$out" "plain HTTP"
contains "empty proxies when asked" "$out" "TRUSTED_PROXIES="
lacks "no proxies trusted" "$out" "TRUSTED_PROXIES=127"

out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/x" --binary "$fake_bin" --port 70000 2>&1)
[ $? != 0 ] && pass "refuses port 70000" || fail "accepted port 70000"
contains "says why" "$out" "between 1 and 65535"

out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/x" --binary "$fake_bin" --port 18082 --session-idle 2h --session-max 1h 2>&1)
[ $? != 0 ] && pass "refuses max shorter than idle" || fail "accepted max shorter than idle"

out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/x" --binary "$fake_bin" --port 18083 --csrf-key short 2>&1)
[ $? != 0 ] && pass "refuses a short key" || fail "accepted a short key"

out=$(bash "$INSTALLER" --dry-run --yes --dir relative --binary "$fake_bin" 2>&1)
[ $? != 0 ] && pass "refuses a relative --dir" || fail "accepted a relative --dir"

out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/x" --binary "$tmp/missing" 2>&1)
[ $? != 0 ] && pass "refuses a missing --binary" || fail "accepted a missing --binary"

mkdir -p "$tmp/existing"
printf 'HOST=127.0.0.1\nPORT=18090\nCSRF_KEY=%s\n' "$(printf 'k%.0s' $(seq 1 40))" >"$tmp/existing/.env"
out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/existing" --binary "$fake_bin" 2>&1)
contains "re-run keeps the settings" "$out" "Keeping its settings"
lacks "re-run writes no .env" "$out" "Writing $tmp/existing/.env"
out=$(bash "$INSTALLER" --dry-run --yes --reconfigure --dir "$tmp/existing" --binary "$fake_bin" 2>&1)
contains "reconfigure keeps the old port as default" "$out" "PORT=18090"
contains "reconfigure keeps the old key" "$out" "CSRF_KEY=<40 characters>"

mkdir -p "$tmp/old/database"
printf 'x' >"$tmp/old/database/passenger.db"
printf 'PORT=8000\nAES_GCM_SECRET=0123456789abcdef0123456789abcdef\n' >"$tmp/old/.env"
out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/old" --binary "$fake_bin" 2>&1)
contains "upgrade is detected" "$out" "old version"
contains "upgrade keeps listening everywhere" "$out" "HOST=0.0.0.0"
contains "upgrade keeps the old port" "$out" "PORT=8000"
contains "upgrade moves the old .env" "$out" "mv $tmp/old/.env $tmp/old/.env.old"
contains "upgrade stops the old service" "$out" "systemctl disable --now passenger-go"
contains "migrate runs as the service user" "$out" "sudo -u passenger $tmp/old/passenger-go migrate --from $tmp/old/database/passenger.db --env $tmp/old/.env.old"
lacks "upgrade does not start the service" "$out" "systemctl enable --now passenger-go"$'\n'
touch "$tmp/old/.env.old"
out=$(bash "$INSTALLER" --dry-run --yes --dir "$tmp/old" --binary "$fake_bin" 2>&1)
[ $? != 0 ] && pass "refuses to overwrite .env.old" || fail "overwrote .env.old"

echo
if [ "$fails" = 0 ]; then
    echo "all install.sh tests passed"
else
    echo "$fails install.sh test(s) failed"
    exit 1
fi
