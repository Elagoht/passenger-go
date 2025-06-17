#!/bin/bash

# TODO: after workflow is done, add a wget command to download latest release from github
cp passenger-go /opt/passenger-go/passenger-go

cp passenger-go.service /etc/systemd/system/passenger-go.service
sed -i "s/{USER}/$(whoami)/g" /etc/systemd/system/passenger-go.service

systemctl daemon-reload
systemctl enable --now passenger-go.service

