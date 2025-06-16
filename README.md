<div align="center">
  <img src="https://raw.githubusercontent.com/Elagoht/passenger-go/refs/heads/main/frontend/static/img/passenger.png" alt="Passenger-Go Logo" width="128" height="128">

# Passenger-Go

</div>

<div align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white" alt="SQLite" />
</div>

Passenger-Go is a secure, self-hosted passphrase manager that runs in a containerized environment. It provides a simple web interface for managing your passwords and sensitive information, with all data encrypted and stored locally on your server.

Additionally provides an api, so you can use it in your own client projects.

## Key Features

- 🔒 AES-GCM encryption for stored data
- 🔑 JWT-based authentication
- 🌐 HTTPS with automatic certificate generation
- 🍃 Environment-based secret management
- 💾 SQLite database for easy backup and portability
- 🐳 Containerized deployment with Docker
- 🐿️ Built with Go for performance and reliability
- 📦 API for client projects (you can create a mobile app, desktop app, etc.)

## Getting Started

1. Clone the repository

```bash
git clone https://github.com/Elagoht/passenger-go.git
```

2. Create a `.env` file with your secrets

```bash
cp .env.example .env
```

then edit the `.env` file with your secrets

3. Build and run the application:

```bash
docker-compose up -d
```

The application will be available at the port you specified in the `.env` file.

## Environment Variables

Required environment variables:

- `DB_PASSPHRASE`: Master encryption key for the database
- `JWT_SECRET`: Secret for JWT token generation
- `AES_GCM_SECRET`: Secret for AES-GCM encryption
- `SALT`: Salt for password hashing

## License

This project is licensed under the [GPL-3.0](LICENSE) license.
