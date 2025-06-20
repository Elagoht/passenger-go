class ApiEndpoint extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: "open" });
    this.isCollapsed = true;
  }

  connectedCallback() {
    const endpoint = JSON.parse(this.getAttribute("data-endpoint"));
    this.render(endpoint);
    this.setupEventListeners();
  }

  setupEventListeners() {
    const toggleButton = this.shadowRoot.querySelector(".toggle-button");
    const header = this.shadowRoot.querySelector(".endpoint-header");
    const content = this.shadowRoot.querySelector(".endpoint-content");

    // Handle toggle button click
    toggleButton.addEventListener("click", (e) => {
      e.stopPropagation();
      this.toggleCollapse(content, toggleButton);
    });

    // Handle header click
    header.addEventListener("click", () => {
      this.toggleCollapse(content, toggleButton);
    });
  }

  toggleCollapse(content, toggleButton) {
    this.isCollapsed = !this.isCollapsed;

    if (this.isCollapsed) {
      content.style.maxHeight = "0";
      content.style.opacity = "0";
      content.style.paddingTop = "0";
      content.style.paddingBottom = "0";
      toggleButton.innerHTML = "▼";
    } else {
      content.style.maxHeight = content.scrollHeight + "px";
      content.style.opacity = "1";
      toggleButton.innerHTML = "▲";
      content.style.paddingTop = "1rem";
      content.style.paddingBottom = "1rem";
    }
  }

  getMethodColor(method) {
    const colors = {
      GET: "#61affe",
      POST: "#49cc90",
      PUT: "#fca130",
      PATCH: "#9b59b6",
      DELETE: "#f93e3e",
    };
    return colors[method] || "#999";
  }

  formatSchema(schema) {
    if (typeof schema === "string") {
      return schema;
    }
    if (Array.isArray(schema)) {
      if (schema.length === 0) return "array";
      if (typeof schema[0] === "string") return "array of strings";
      if (typeof schema[0] === "object") return "array of objects";
      return "array";
    }
    if (typeof schema === "object") {
      const keys = Object.keys(schema);
      if (keys.length === 0) return "object";
      return `{ ${keys.map((key) => `${key}: ${schema[key]}`).join(", ")} }`;
    }
    return "object";
  }

  formatExample(example) {
    if (typeof example === "string") {
      return example;
    }
    if (Array.isArray(example)) {
      return JSON.stringify(example, null, 2);
    }
    if (typeof example === "object") {
      return JSON.stringify(example, null, 2);
    }
    return example;
  }

  getRequirementBadges(endpoint) {
    const badges = [];

    if (endpoint.init) {
      badges.push({ text: "Init Required", color: "#e67e22" });
    }

    if (endpoint.auth) {
      badges.push({ text: "Auth Required", color: "#e74c3c" });
    } else {
      badges.push({ text: "Public", color: "#27ae60" });
    }

    return badges;
  }

  getContentTypeBadges(endpoint) {
    const badges = [];

    if (endpoint.request && endpoint.request.type) {
      badges.push({ text: `Req: ${endpoint.request.type}`, color: "#3498db" });
    }
    if (endpoint.response && endpoint.response.type) {
      badges.push({ text: `Res: ${endpoint.response.type}`, color: "#9b59b6" });
    }

    return badges;
  }

  render(endpoint) {
    const methodColor = this.getMethodColor(endpoint.method);
    const requirementBadges = this.getRequirementBadges(endpoint);
    const collapsedStyle = this.isCollapsed
      ? "max-height:0;opacity:0;"
      : "max-height:none;opacity:1;";
    const toggleIcon = this.isCollapsed ? "▼" : "▲";

    this.shadowRoot.innerHTML = `
      <style>
        :host {
          display: block;
          background-color: #1e1e2e;
          border: 0.06125rem solid #45475a;
          border-radius: 0.5rem;
          overflow: hidden;
          font-family: inherit;
        }

        .endpoint-header {
          display: flex;
          align-items: center;
          gap: 0.75rem;
          padding: 1rem;
          border-bottom: 0.06125rem solid #45475a;
          background-color: #313244;
          cursor: pointer;
          user-select: none;
        }

        .endpoint-header:hover {
          background-color: #3a3a4a;
        }

        .method-badge {
          padding: 0.25rem 0.5rem;
          border-radius: 0.25rem;
          font-weight: 600;
          font-size: 0.75rem;
          text-transform: uppercase;
          color: white;
          min-width: 3rem;
          text-align: center;
        }

        .path {
          font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
          font-size: 0.875rem;
          color: #cdd6f4;
          flex: 1;
        }

        .requirement-badges {
          display: flex;
          gap: 0.5rem;
        }

        .requirement-badge {
          padding: 0.25rem 0.5rem;
          border-radius: 0.25rem;
          font-size: 0.75rem;
          font-weight: 500;
          color: white;
        }

        .toggle-button {
          background: none;
          border: none;
          color: #cdd6f4;
          font-size: 1rem;
          cursor: pointer;
          padding: 0.25rem;
          border-radius: 0.25rem;
          transition: background-color 0.2s ease;
          display: flex;
          align-items: center;
          justify-content: center;
          min-width: 1.5rem;
          height: 1.5rem;
        }

        .toggle-button:hover {
          background-color: #45475a;
        }

        .endpoint-content {
          padding: 0 1rem;
          opacity: 1;
          transition: all 0.3s ease-in-out;
          overflow: hidden;
        }

        .description {
          margin-bottom: 1rem;
          color: #cdd6f4;
          line-height: 1.5;
        }

        .sections {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(20rem, 1fr));
          gap: 1rem;
        }

        .section {
          background-color: #313244;
          border: 0.06125rem solid #45475a;
          border-radius: 0.25rem;
          padding: 1rem;
        }

        .section h4 {
          margin: 0 0 0.75rem 0;
          font-size: 0.875rem;
          font-weight: 600;
          color: #a6adc8;
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .schema {
          font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
          font-size: 0.75rem;
          background-color: #1e1e2e;
          padding: 0.5rem;
          border-radius: 0.25rem;
          border: 0.06125rem solid #45475a;
          color: #cdd6f4;
          white-space: pre-wrap;
          word-break: break-word;
        }

        .status-codes {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
        }

        .status-code {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.25rem 0.5rem;
          border-radius: 0.25rem;
          background-color: #1e1e2e;
          border: 0.06125rem solid #45475a;
        }

        .status-number {
          font-weight: 600;
          font-size: 0.75rem;
          min-width: 2.5rem;
          text-align: center;
          padding: 0.125rem 0.25rem;
          border-radius: 0.125rem;
          color: white;
        }

        .status-200 { background-color: #27ae60; }
        .status-201 { background-color: #27ae60; }
        .status-204 { background-color: #27ae60; }
        .status-400 { background-color: #e74c3c; }
        .status-401 { background-color: #e74c3c; }
        .status-404 { background-color: #e74c3c; }
        .status-409 { background-color: #f39c12; }
        .status-412 { background-color: #f39c12; }
        .status-422 { background-color: #e74c3c; }
        .status-500 { background-color: #e74c3c; }

        .status-description {
          font-size: 0.75rem;
          color: #cdd6f4;
        }

        .no-content {
          color: #a6adc8;
          font-style: italic;
          font-size: 0.875rem;
        }

        .status-codes-section {
          grid-column: span 2;
        }

        @media (max-width: 48rem) {
          .sections {
            grid-template-columns: 1fr;
          }
          
          .endpoint-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 0.5rem;
          }
          
          .path {
            word-break: break-all;
          }
          
          .requirement-badges {
            flex-wrap: wrap;
          }
        }

        .endpoint-full {
          display: flex;
          align-items: center;
          gap: 0.75rem;
          flex: 1;
        }

        .content-type {
          font-size: 0.75rem;
          font-weight: 500;
          color: #a6adc8;
          opacity: 0.8;
        }

        .content-type-badge {
          padding: 0.25rem 0.5rem;
          border-radius: 0.25rem;
          font-size: 0.75rem;
          font-weight: 500;
          color: white;
          background-color: #3498db;
        }
      </style>

      <div class="endpoint-header">
        <div class="endpoint-full">
          <div class="method-badge" style="background-color: ${methodColor}">
            ${endpoint.method}
          </div>
          <div class="path">${endpoint.path}</div>
        </div>
        <div class="requirement-badges">
          ${requirementBadges
            .map(
              (badge) => `
            <div class="requirement-badge" style="background-color: ${badge.color}">
              ${badge.text}
            </div>
          `
            )
            .join("")}
          <button class="toggle-button" type="button">${toggleIcon}</button>
        </div>
      </div>

      <div class="endpoint-content" style="${collapsedStyle}">
        <div class="description">${endpoint.description}</div>
        
        <div class="sections">
          ${
            endpoint.request
              ? `
            <div class="section">
              <h4>Request ${
                endpoint.request.type
                  ? `<span class="content-type">(${endpoint.request.type})</span>`
                  : ""
              }</h4>
              ${
                endpoint.request.type === "multipart/form-data"
                  ? `
                <div class="schema">${endpoint.request.schema.file}</div>
              `
                  : endpoint.request.type === "application/json" ||
                    endpoint.request.type === "application/octet-stream"
                  ? `
                <div class="schema">${this.formatSchema(
                  endpoint.request.schema
                )}</div>
                <h4 style="margin-top: 0.75rem; margin-bottom: 0.5rem;">Example</h4>
                <div class="schema">${this.formatExample(
                  endpoint.request.example
                )}</div>
              `
                  : `
                <div class="schema">${endpoint.request.type}</div>
              `
              }
            </div>
          `
              : `
            <div class="section">
              <h4>Request</h4>
              <div class="no-content">No request body required</div>
            </div>
          `
          }

          <div class="section">
            <h4>Response ${
              endpoint.response && endpoint.response.type
                ? `<span class="content-type">(${endpoint.response.type})</span>`
                : ""
            }</h4>
            ${
              !endpoint.response
                ? `
              <div class="schema">No response body</div>
            `
                : endpoint.response.type === "text/csv"
                ? `
              <div class="schema">${endpoint.response.schema}</div>
            `
                : endpoint.response.type === "application/json" ||
                  endpoint.response.type === "application/octet-stream"
                ? `
              ${
                Array.isArray(endpoint.response.schema)
                  ? `
                <div class="schema">Array of ${this.formatSchema(
                  endpoint.response.schema[0]
                )}</div>
              `
                  : typeof endpoint.response.schema === "string"
                  ? `
                <div class="schema">${endpoint.response.schema}</div>
              `
                  : `
                <div class="schema">${this.formatSchema(
                  endpoint.response.schema
                )}</div>
              `
              }
              <h4 style="margin-top: 0.75rem; margin-bottom: 0.5rem;">Example</h4>
              <div class="schema">${this.formatExample(
                endpoint.response.example
              )}</div>
            `
                : `
              <div class="schema">${endpoint.response.schema}</div>
            `
            }
          </div>

          <div class="section status-codes-section">
            <h4>Status Codes</h4>
            <div class="status-codes">
              ${endpoint.statusCodes
                .map(
                  (status) => `
                <div class="status-code">
                  <div class="status-number status-${status.code}">${status.code}</div>
                  <div class="status-description">${status.description}</div>
                </div>
              `
                )
                .join("")}
            </div>
          </div>
        </div>
      </div>
    `;
  }
}

customElements.define("api-endpoint", ApiEndpoint);
