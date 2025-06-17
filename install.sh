#!/bin/bash

# If not linux, exit
if [ "$(uname -s)" != "Linux" ]; then
    echo "This script is only for Linux"
    exit 1
fi

# TODO: after workflow is done, add a wget command to download latest release from github
curl -sL "https://github.com/Elagoht/passenger-go/releases/latest/download/passenger-go-linux-$(uname -m)" -o /opt/passenger-go/passenger-go

chmod +x /opt/passenger-go/passenger-go

cp passenger-go.service /etc/systemd/system/passenger-go.service
sed -i "s/{USER}/$(whoami)/g" /etc/systemd/system/passenger-go.service

systemctl daemon-reload
systemctl enable --now passenger-go.service