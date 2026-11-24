#!/bin/bash

infoPrint() {
    echo -e "\033[1;32m$1\033[0m"
}

errorPrint() {
    echo -e "\033[1;31m$1\033[0m"
}

successPrint() {
    echo -e "\033[1;32m$1\033[0m"
}

INSTALL_DIR=/opt/passenger-go
OLD_DB="$INSTALL_DIR/database/passenger.db"
NEW_DB="$INSTALL_DIR/data/passenger.db"

infoPrint "Installing Passenger Go..."

infoPrint "Checking architecture..."
arch=$(uname -m)
if [ "$arch" == "x86_64" ]; then
    arch="amd64"
elif [ "$arch" == "aarch64" ]; then
    arch="arm64"
else
    errorPrint "Unsupported architecture: $arch"
    exit 1
fi

infoPrint "Architecture: $arch"

# If not linux, exit
if [ "$(uname -s)" != "Linux" ]; then
    errorPrint "This script is only for Linux"
    exit 1
fi

if ! command -v openssl >/dev/null 2>&1; then
    errorPrint "openssl is required to generate CSRF_KEY"
    exit 1
fi

# An old installation has a database in database/ and an .env holding
# AES_GCM_SECRET. Never clobber a previously preserved .env.old.
upgrade=false
if [ -f "$OLD_DB" ]; then
    upgrade=true
    if [ -e "$INSTALL_DIR/.env.old" ]; then
        errorPrint "$INSTALL_DIR/.env.old already exists. Refusing to overwrite it."
        errorPrint "It may hold your old AES_GCM_SECRET. If a previous run of this script already moved your old .env there,"
        errorPrint "finish the migration steps it printed (stop the service, run migrate, delete the old files), or move"
        errorPrint ".env.old back to $INSTALL_DIR/.env by hand, then run this script again."
        exit 1
    fi
    if [ ! -f "$INSTALL_DIR/.env" ]; then
        errorPrint "Old database found but $INSTALL_DIR/.env is missing; the old AES_GCM_SECRET is needed to migrate."
        exit 1
    fi
    # Stop the old service so the new binary cannot be started (Restart=always)
    # and create data/passenger.db before migrate runs.
    infoPrint "Stopping the old service..."
    sudo systemctl stop passenger-go || true
fi

# Re-run on a new-format install: keep the existing configuration untouched.
keep_env=false
if [ "$upgrade" == false ] && [ -f "$INSTALL_DIR/.env" ]; then
    keep_env=true
    infoPrint "Existing $INSTALL_DIR/.env found; keeping it unchanged."
fi

infoPrint "Creating directory..."
sudo mkdir -p "$INSTALL_DIR"

infoPrint "Downloading latest release..."
# Download to a temporary file and rename, so a running old binary is not overwritten in place.
if ! sudo curl -fsSL "https://github.com/Elagoht/passenger-go/releases/latest/download/passenger-go-linux-$arch" -o "$INSTALL_DIR/passenger-go.new"; then
    errorPrint "Download failed"
    sudo rm -f "$INSTALL_DIR/passenger-go.new"
    exit 1
fi
sudo chmod +x "$INSTALL_DIR/passenger-go.new"
sudo mv -f "$INSTALL_DIR/passenger-go.new" "$INSTALL_DIR/passenger-go"

infoPrint "Setting permissions..."
sudo chown -R "$(whoami)" "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data"
chmod 700 "$INSTALL_DIR/data"
successPrint "Permissions set"

# Preserve the old .env BEFORE writing the new one.
if [ "$upgrade" == true ]; then
    infoPrint "Old installation found. Moving old .env to .env.old..."
    mv "$INSTALL_DIR/.env" "$INSTALL_DIR/.env.old" || exit 1
    chmod 600 "$INSTALL_DIR/.env.old"
fi

infoPrint "Copying service file..."
sudo cp passenger-go.service /etc/systemd/system/passenger-go.service
sudo sed -i "s/{USER}/$(whoami)/g" /etc/systemd/system/passenger-go.service

if [ "$keep_env" == false ]; then
    infoPrint "If you will set a custom PORT, write it here."
    read -rp "PORT: " port
    if [ -z "$port" ]; then
        port="8080"
    fi
    case "$port" in
        '' | *[!0-9]*)
            errorPrint "PORT must be a number between 1 and 65535"
            exit 1
            ;;
    esac
    if [ "$((10#$port))" -lt 1 ] || [ "$((10#$port))" -gt 65535 ]; then
        errorPrint "PORT must be a number between 1 and 65535"
        exit 1
    fi

    infoPrint "Writing $INSTALL_DIR/.env..."
    csrf_key=$(openssl rand -hex 32) || exit 1
    (
        umask 077
        cat > "$INSTALL_DIR/.env" <<ENVFILE
HOST=0.0.0.0
PORT=$port
DATABASE_PATH=$NEW_DB
CSRF_KEY=$csrf_key
ENVFILE
    )
    chmod 600 "$INSTALL_DIR/.env"
    successPrint "PORT set to $port"
fi

infoPrint "Reloading systemd..."
sudo systemctl daemon-reload

if [ "$upgrade" == true ]; then
    errorPrint "An old installation was found. The service has NOT been enabled or started."
    echo
    echo "Migrate your accounts into the new vault, in this order:"
    echo
    echo "  1. Stop the old service:"
    echo "       sudo systemctl stop passenger-go"
    echo "  2. Run the migration (your master passphrase stays the same):"
    echo "       $INSTALL_DIR/passenger-go migrate --from $OLD_DB --env $INSTALL_DIR/.env.old --to $NEW_DB"
    echo "  3. Write down the NEW recovery key it prints; the old one no longer works."
    echo "  4. After a successful migration, delete the old files (they hold the old secret):"
    echo "       rm -f $OLD_DB $INSTALL_DIR/.env.old"
    echo "  5. Enable and start the service:"
    echo "       sudo systemctl enable --now passenger-go.service"
    exit 1
fi

if [ "$keep_env" == true ] && sudo systemctl is-enabled --quiet passenger-go.service; then
    infoPrint "Restarting service..."
    sudo systemctl restart passenger-go.service
else
    infoPrint "Enabling and starting service..."
    sudo systemctl enable --now passenger-go.service
fi

successPrint "Done!"
if [ "$keep_env" == false ]; then
    infoPrint "Passenger Go prints a one-time setup code on first start. Find it with:"
    echo "    journalctl -u passenger-go | grep \"setup code\""
fi
