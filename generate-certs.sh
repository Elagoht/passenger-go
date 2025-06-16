#!/bin/bash

# Create certs directory if it doesn't exist
mkdir -p certs

# Generate private key
openssl genrsa -out certs/server.key 2048

# Generate CSR (Certificate Signing Request)
openssl req -new -key certs/server.key -out certs/server.csr -subj "/CN=localhost"

# Generate self-signed certificate
openssl x509 -req -days 365 -in certs/server.csr -signkey certs/server.key -out certs/server.crt

# Set proper permissions
chmod 600 certs/server.key
chmod 644 certs/server.crt

echo "Self-signed certificates generated in the 'certs' directory" 