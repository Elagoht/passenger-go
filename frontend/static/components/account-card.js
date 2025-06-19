class AccountCard extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: "open" });
  }

  static get observedAttributes() {
    return ["data-account", "data-query"];
  }

  connectedCallback() {
    this.render();
  }

  attributeChangedCallback() {
    this.render();
  }

  getInitials(text) {
    return text
      .split(" ")
      .map((word) => word.charAt(0))
      .join("")
      .toUpperCase()
      .slice(0, 2);
  }

  extractDomain(url) {
    if (!url) return "";
    try {
      const urlObj = new URL(url.startsWith("http") ? url : `https://${url}`);
      return urlObj.hostname.replace(/^www\./, "");
    } catch {
      return url.split("/")[0].split(":")[0];
    }
  }

  getStrengthColor(strength) {
    const colors = {
      0: "#f38ba8",
      1: "#fab387",
      2: "#f9e2af",
      3: "#a6e3a1",
      4: "#94e2d5",
    };
    if (strength > 4) strength = 4;
    if (strength < 0) strength = 0;
    return colors[strength];
  }

  highlightText(text, query) {
    if (!query) return text;
    const regex = new RegExp(`(${query})`, "gi");
    return text.replace(regex, "<mark>$1</mark>");
  }

  handleFaviconError(img, platform) {
    const initials = this.getInitials(platform);
    const container = img.parentElement;
    container.innerHTML = `<div class="initials">${initials}</div>`;
  }

  sanitizeUrl(url) {
    if (!url.startsWith("http")) url = `https://${url}`;
    return url;
  }

  render() {
    const accountData = this.getAttribute("data-account");
    const query = this.getAttribute("data-query") || "";

    if (!accountData) return;

    const account = JSON.parse(accountData);
    const domain = this.extractDomain(account.url);
    const strengthColor = this.getStrengthColor(account.strength);

    this.shadowRoot.innerHTML = `
      <style>
        .card {
          background-color: #1e1e2e;
          border: 0.06125rem solid #45475a;
          border-radius: 0.5rem;
          display: flex;
          flex-direction: column;
          gap: 0.75rem;
          padding: 0.75rem;
          transition: all 0.2s ease;
          position: relative;
          overflow: hidden;
        }

        .card:hover {
          background-color: #262637;
          border-color: #585b70;
        }

        .strength-indicator {
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          height: 0.1875rem;
          background-color: ${strengthColor};
        }

        .card-header {
          display: flex;
          align-items: center;
          gap: 0.75rem;
          cursor: pointer;
          border-radius: 0.375rem;
          padding: 0.5rem;
          transition: background-color 0.2s ease;
        }

        .card-header:hover {
          background-color: rgba(255, 255, 255, 0.05);
        }

        .favicon-container {
          width: 4rem;
          height: 4rem;
          border-radius: 0.5rem;
          background-color: #313244;
          display: flex;
          align-items: center;
          justify-content: center;
          overflow: hidden;
          flex-shrink: 0;
        }

        .favicon {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }

        .initials {
          color: #cdd6f4;
          font-weight: 600;
          font-size: 1.125rem;
        }

        .card-info {
          flex: 1;
          min-width: 0;
        }

        .platform {
          font-weight: 600;
          font-size: 1rem;
          color: #cdd6f4;
          margin-bottom: 0.25rem;
          word-wrap: break-word;
        }

        .identifier {
          font-size: 0.875rem;
          color: #a6adc8;
          word-wrap: break-word;
        }

        .card-actions {
          display: flex;
          gap: 0.5rem;
        }

        .btn {
          background-color: #666baa;
          color: #cdd6f4;
          border: none;
          border-radius: 0.25rem;
          font-size: 0.875rem;
          cursor: pointer;
          transition: all 0.2s ease;
          text-decoration: none;
          text-align: center;
          display: flex;
          align-items: center;
          justify-content: center;
          min-height: 2.25rem;
        }

        .btn:hover {
          background-color: #8589cf;
        }

        .btn-primary {
          flex: 1;
        }

        .btn-secondary {
          background-color: #45475a;
          color: #cdd6f4;
          width: 2.25rem;
        }

        .btn-secondary:hover {
          background-color: #585b70;
        }

        mark {
          background-color: #f9e2af;
          color: #1e1e2e;
          border-radius: 0.125rem;
        }

        .external-link {
          text-decoration: none;
          color: inherit;
        }
      </style>

      <div class="card">
        <div class="strength-indicator"></div>
        <div class="card-header" onclick="this.getRootNode().host.navigateToDetails(${
          account.id
        })">
          <div class="favicon-container">
            <img 
              class="favicon" 
              src="https://icon.horse/icon/${domain}" 
              alt="${account.platform} favicon"
              onerror="this.getRootNode().host.handleFaviconError(this, '${
                account.platform
              }')"
            />
          </div>
          <div class="card-info">
            <div class="platform">${this.highlightText(
              account.platform,
              query
            )}</div>
            <div class="identifier">${this.highlightText(
              account.identifier,
              query
            )}</div>
          </div>
        </div>
        <div class="card-actions">
          <button class="btn btn-primary" onclick="this.getRootNode().host.copyPassphrase(${
            account.id
          })">
            Copy Password
          </button>
          <button class="btn btn-secondary" onclick="this.getRootNode().host.copyIdentifier('${
            account.identifier
          }')" title="Copy Username">
            👤
          </button>
          <a class="btn btn-secondary external-link" href="${this.sanitizeUrl(
            account.url
          )}" target="_blank" rel="noopener noreferrer" title="Open URL">
            🔗
          </a>
        </div>
      </div>
    `;
  }

  copyPassphrase(id) {
    this.dispatchEvent(
      new CustomEvent("copy-passphrase", {
        detail: { id },
        bubbles: true,
      })
    );
  }

  copyIdentifier(identifier) {
    this.dispatchEvent(
      new CustomEvent("copy-text", {
        detail: { text: identifier },
        bubbles: true,
      })
    );
  }

  navigateToDetails(id) {
    window.location.href = `/accounts/${id}`;
  }
}

customElements.define("account-card", AccountCard);
