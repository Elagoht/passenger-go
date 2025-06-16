# Build stage
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cli

# Final stage
FROM alpine:latest

WORKDIR /app

# Install required packages
RUN apk --no-cache add ca-certificates openssl

# Copy the binary and startup script
COPY --from=builder /app/main .
COPY start.sh .

# Make startup script executable
RUN chmod +x start.sh

# Copy .env file
COPY --from=builder /app/.env .

# Copy frontend files
COPY --from=builder /app/frontend/templates /app/frontend/templates
COPY --from=builder /app/frontend/static /app/frontend/static

# Expose HTTPS port
EXPOSE 443

# Run the startup script
CMD ["./start.sh"]