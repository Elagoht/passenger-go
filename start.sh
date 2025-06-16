#!/bin/sh

# Generate self-signed certificates
mkdir -p /app/certs

# Create OpenSSL config
cat > /app/certs/openssl.cnf << EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ID
ST = State
L = City
O = Organization
OU = Unit
CN = localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
EOF

# Generate private key and CSR
openssl req -new -newkey rsa:2048 -nodes \
    -keyout /app/certs/server.key \
    -out /app/certs/server.csr \
    -config /app/certs/openssl.cnf

# Generate self-signed certificate
openssl x509 -req -days 365 \
    -in /app/certs/server.csr \
    -signkey /app/certs/server.key \
    -out /app/certs/server.crt \
    -extensions v3_req \
    -extfile /app/certs/openssl.cnf

# Set proper permissions
chmod 600 /app/certs/server.key
chmod 644 /app/certs/server.crt

# Clean up
rm /app/certs/server.csr /app/certs/openssl.cnf

# Start the application
exec /app/main