<div align="center">
  <img src="https://raw.githubusercontent.com/Elagoht/passenger-go/refs/heads/main/frontend/static/img/passenger.png" alt="Passenger-Go Logo" width="128" height="128">

# Passenger-Go

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white)
![HTML](https://img.shields.io/badge/HTML-E34F26?style=for-the-badge&logo=html5&logoColor=white)
![CSS](https://img.shields.io/badge/CSS-1572B6?style=for-the-badge&logo=css&logoColor=white)
![JavaScript](https://img.shields.io/badge/JavaScript-F7DF1E?style=for-the-badge&logo=javascript&logoColor=black)
![Tailscale](https://img.shields.io/badge/Tailscale-000000?style=for-the-badge&logo=tailscale&logoColor=white)
---

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/Elagoht/passenger-go/release.yaml?style=for-the-badge)
[![GitHub Stars](https://img.shields.io/github/stars/Elagoht/passenger-go.svg?style=for-the-badge)](https://github.com/Elagoht/passenger-go/stargazers)
![GitHub License](https://img.shields.io/github/license/Elagoht/passenger-go?style=for-the-badge)
</div>

Passenger-Go is a secure, self-hosted passphrase manager that runs in your local network. It provides a modern, responsive web interface for managing your passphrases and sensitive information, with all data encrypted and stored locally on your server.

Additionally provides an api, so you can use it in your own client projects.

## Recommended Way to Run

Passenger-Go is designed to run as a always online server. So you can consider using a VPS or a dedicated server.

You can use [Tailscale](https://tailscale.com) to connect to your server from anywhere.

### Prerequisites

- [Tailscale](https://tailscale.com)

1. **Install Tailscale:** Assuming you are using a Linux server, and you already installed and configured Tailscale.
2. **Clone the repository:**

```bash
git clone https://github.com/Elagoht/passenger-go.git
cd passenger-go
```

3. **Run the installation script:**

```bash
./install.sh
```

> The installation script will ask you for the following information:
>
> - The port to run the server on (default: 8080)
> - The JWT secret (a complex string)
> - The AES-GCM secret (exactly 32 characters)
> - The salt (recommended: 16 characters)

4. **Serve the application with Tailscale:**

```bash
tailscale serve --port http://localhost:[YOUR_PORT_HERE]
```

5. **Use the application!**: Your server is now accessible only in your Tailscale network. Your tailscale domain will be your server's domain.

## Key Features

- 🔒 AES-GCM encryption for stored data
- 🔑 JWT-based authentication
- 🍃 Environment-based secret management
- 💾 SQLite database for easy backup and portability
- 🐿️ Built with Go for performance and reliability
- 🎨 Modern, responsive UI with dark/light mode support
- 🔍 Real-time search functionality
- 🌐 Automatic favicon fetching for account cards
- 📱 Mobile-friendly design
- 📦 API for client projects (you can create a mobile app, desktop app, etc.)

## UI Features

- **Modern Card Layout**: Clean, card-based interface for easy account management
- **Favicon Support**: Automatic favicon fetching from websites using icon.horse
- **URL Integration**: Click to open account websites in new tabs
- **Real-time Search**: Instant search across platform names, usernames, and notes
- **Passphrase Generator**: Generate strong passphrases with customizable length and complexity
- **Passphrase Alternator**: Changes the given passphrase's characters to similar looking characters
- **Responsive Design**: Works perfectly on desktop, tablet, and mobile devices
- **Copy to Clipboard**: One-click copying of usernames and passphrases
- **Import/Export**: Support for Firefox and Chromium CSV exports
- **API Documentation**: Comprehensive API reference with interactive endpoint documentation

## Environment Variables

Required environment variables:

- `JWT_SECRET`: Secret for JWT token generation
- `AES_GCM_SECRET`: Secret for AES-GCM encryption
- `SALT`: Salt for passphrase hashing
- `PORT`: Port to run the server on

## License

This project is licensed under the [GPL-3.0](LICENSE) license.

## Open Source

This project is open source and free to use. Feel free to contribute to the project.

## Why Not Saying "Password"

Passwords are obsolete and not secure. Passphrases are more secure and easier to remember. word means, a single word. phrase means, a group of words. How long is your "passkey" means how secure it is. By using phrases, you can remember them easier while being more secure.
