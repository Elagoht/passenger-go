#!/bin/bash

echo "Installing Passenger Go..."

echo "Checking architecture..."
arch=$(uname -m)
if [ "$arch" == "x86_64" ]; then
    arch="amd64"
elif [ "$arch" == "aarch64" ]; then
    arch="arm64"
else
    echo "Unsupported architecture: $arch"
    exit 1
fi

echo "Architecture: $arch"

# If not linux, exit
if [ "$(uname -s)" != "Linux" ]; then
    echo "This script is only for Linux"
    exit 1
fi

echo "Creating directory..."
sudo mkdir -p /opt/passenger-go

echo "Downloading latest release..."
sudo curl -sL "https://github.com/Elagoht/passenger-go/releases/latest/download/passenger-go-linux-$arch" -o /opt/passenger-go/passenger-go

echo "Setting permissions..."
sudo chmod +x /opt/passenger-go/passenger-go
sudo chown -R $(whoami) /opt/passenger-go
echo "Permissions set"

echo "Copying service file..."
sudo cp passenger-go.service /etc/systemd/system/passenger-go.service
sudo sed -i "s/{USER}/$(whoami)/g" /etc/systemd/system/passenger-go.service

echo "If you will set a custom PORT, write it here."
read -p "PORT: " port
if [ -z "$port" ]; then
    port="8080"
fi

if [ -n "$port" ]; then
    sudo sed -i "s/{PORT}/$port/g" /etc/systemd/system/passenger-go.service
    echo "PORT=$port" >> /opt/passenger-go/.env
    echo "PORT set to $port"
else
    echo "Default PORT will be used"
fi

echo Now we need this ENV variables:
read -p "JWT_SECRET: " jwt_secret
echo "JWT_SECRET=$jwt_secret" >> /opt/passenger-go/.env
read -p "AES_GCM_SECRET: " aes_gcm_secret
echo "AES_GCM_SECRET=$aes_gcm_secret" >> /opt/passenger-go/.env
read -p "SALT: " salt
echo "SALT=$salt" >> /opt/passenger-go/.env

echo "Reloading systemd..."
sudo systemctl daemon-reload

echo "Enabling and starting service..."
sudo systemctl enable --now passenger-go.service

echo "Done!"