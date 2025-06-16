#!/bin/sh

# Read PORT from .env file if it exists
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Set default port if not provided in .env
if [ -z "$PORT" ]; then
    export PORT=1234
fi

# Generate self-signed certificates
mkdir -p /app/certs

# Create OpenSSL config with proper SAN configuration
cat > /app/certs/openssl.cnf << EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
C = US
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
IP.2 = ::1
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

# Set SSL certificate paths in environment
export SSL_CERT_PATH=/app/certs/server.crt
export SSL_KEY_PATH=/app/certs/server.key

# Print configuration for debugging
echo "Starting server with configuration:"
echo "PORT: $PORT"
echo "SSL_CERT_PATH: $SSL_CERT_PATH"
echo "SSL_KEY_PATH: $SSL_KEY_PATH"

# Verify certificates exist and are readable
if [ ! -f "$SSL_CERT_PATH" ] || [ ! -f "$SSL_KEY_PATH" ]; then
    echo "Error: SSL certificates not found!"
    exit 1
fi

# Start the application
exec /app/main