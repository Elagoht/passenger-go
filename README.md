<div align="center">
  <img src="https://raw.githubusercontent.com/Elagoht/passenger-go/refs/heads/main/frontend/static/img/passenger.png" alt="Passenger-Go Logo" width="128" height="128">

# Passenger-Go

</div>

<div align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white" alt="SQLite" />
</div>

Passenger-Go is a secure, self-hosted passphrase manager that runs in a containerized environment. It provides a simple web interface for managing your passwords and sensitive information, with all data encrypted and stored locally on your server.

Additionally provides an api, so you can use it in your own client projects.

## Key Features

- 🔒 AES-GCM encryption for stored data
- 🔑 JWT-based authentication
- 🍃 Environment-based secret management
- 💾 SQLite database for easy backup and portability
- 🐿️ Built with Go for performance and reliability
- 📦 API for client projects (you can create a mobile app, desktop app, etc.)

## Environment Variables

Required environment variables:

- `JWT_SECRET`: Secret for JWT token generation
- `AES_GCM_SECRET`: Secret for AES-GCM encryption
- `SALT`: Salt for password hashing
- `PORT`: Port to run the server on

## License

This project is licensed under the [GPL-3.0](LICENSE) license.
