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

infoPrint "Creating directory..."
sudo mkdir -p /opt/passenger-go

infoPrint "Downloading latest release..."
sudo curl -sL "https://github.com/Elagoht/passenger-go/releases/latest/download/passenger-go-linux-$arch" -o /opt/passenger-go/passenger-go

infoPrint "Copying templates..."
sudo mkdir -p /opt/passenger-go/frontend/templates
sudo cp -r ./frontend/templates /opt/passenger-go/frontend
infoPrint "Copying static files..."
sudo mkdir -p /opt/passenger-go/frontend/static
sudo cp -r ./frontend/static /opt/passenger-go/frontend

infoPrint "Setting permissions..."
sudo chmod +x /opt/passenger-go/passenger-go
sudo chown -R $(whoami) /opt/passenger-go
successPrint "Permissions set"

infoPrint "Copying service file..."
sudo cp passenger-go.service /etc/systemd/system/passenger-go.service
sudo sed -i "s/{USER}/$(whoami)/g" /etc/systemd/system/passenger-go.service

infoPrint "If you will set a custom PORT, write it here."
read -p "PORT: " port
if [ -z "$port" ]; then
    port="8080"
fi

if [ -n "$port" ]; then
    sudo sed -i "s/{PORT}/$port/g" /etc/systemd/system/passenger-go.service
    echo "PORT=$port" >> /opt/passenger-go/.env
    successPrint "PORT set to $port"
else
    infoPrint "Default PORT will be used"
fi

infoPrint "Now we need this ENV variables:"
read -p "JWT_SECRET: " jwt_secret
echo "JWT_SECRET=$jwt_secret" >> /opt/passenger-go/.env
read -p "AES_GCM_SECRET: " aes_gcm_secret
echo "AES_GCM_SECRET=$aes_gcm_secret" >> /opt/passenger-go/.env
read -p "SALT: " salt
echo "SALT=$salt" >> /opt/passenger-go/.env

infoPrint "Reloading systemd..."
sudo systemctl daemon-reload

infoPrint "Enabling and starting service..."
sudo systemctl enable --now passenger-go.service

successPrint "Done!"