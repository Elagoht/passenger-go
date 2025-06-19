class URLCompletion {
  constructor(inputElement, datalistElement) {
    this.input = inputElement;
    this.datalist = datalistElement;
    this.popularTLDs = [
      ".com",
      ".net",
      ".org",
      ".edu",
      ".gov",
      ".mil",
      ".int",
      ".co",
      ".io",
      ".dev",
      ".app",
      ".tech",
      ".info",
      ".biz",
      ".me",
      ".tv",
      ".cc",
      ".ly",
      ".ai",
      ".cloud",
      ".online",
    ];
    this.init();
  }

  init() {
    this.input.addEventListener("input", this.handleInput.bind(this));
    this.input.addEventListener("blur", this.normalizeURL.bind(this));
  }

  handleInput(event) {
    const value = event.target.value.trim();
    if (value.length < 2) {
      this.clearDatalist();
      return;
    }

    const suggestions = this.generateSuggestions(value);
    this.updateDatalist(suggestions);
  }

  generateSuggestions(input) {
    const suggestions = new Set();
    const cleanInput = this.cleanInput(input);

    // Add protocol variants if not present
    if (
      !cleanInput.startsWith("http://") &&
      !cleanInput.startsWith("https://")
    ) {
      suggestions.add(`https://${cleanInput}`);
      suggestions.add(`http://${cleanInput}`);
    }

    // Generate TLD suggestions
    const tldSuggestions = this.generateTLDSuggestions(input); // Use original input, not cleanInput
    tldSuggestions.forEach((suggestion) => suggestions.add(suggestion));

    // Add normalized versions
    const normalized = this.normalizeURLString(cleanInput);
    if (normalized !== cleanInput) {
      suggestions.add(normalized);
      if (
        !normalized.startsWith("http://") &&
        !normalized.startsWith("https://")
      ) {
        suggestions.add(`https://${normalized}`);
        suggestions.add(`http://${normalized}`);
      }
    }

    return Array.from(suggestions).slice(0, 10); // Limit to 10 suggestions
  }

  generateTLDSuggestions(input) {
    const suggestions = [];
    const hasProtocol =
      input.startsWith("http://") || input.startsWith("https://");
    const domain = hasProtocol ? input.replace(/^https?:\/\//, "") : input;

    // If domain doesn't have a TLD or has an incomplete one
    const parts = domain.split(".");
    const lastPart = parts[parts.length - 1];

    // If no TLD or incomplete TLD
    if (
      parts.length === 1 ||
      (lastPart.length < 2 && !lastPart.includes("/"))
    ) {
      const baseDomain = parts[0];
      this.popularTLDs.forEach((tld) => {
        const suggestion = hasProtocol
          ? input.replace(/^(https?:\/\/).*/, `$1${baseDomain}${tld}`)
          : `${baseDomain}${tld}`;
        suggestions.push(suggestion);
      });
    }

    // If partial TLD match
    else if (lastPart.length < 4 && !lastPart.includes("/")) {
      const baseDomain = parts.slice(0, -1).join(".");
      const partialTLD = `.${lastPart}`;

      this.popularTLDs
        .filter((tld) => tld.startsWith(partialTLD))
        .forEach((tld) => {
          const suggestion = hasProtocol
            ? input.replace(/^(https?:\/\/).*/, `$1${baseDomain}${tld}`)
            : `${baseDomain}${tld}`;
          suggestions.push(suggestion);
        });
    }

    return suggestions;
  }

  cleanInput(input) {
    // Remove extra whitespace
    let cleaned = input.trim();

    // Remove trailing slashes except for protocol
    if (cleaned.endsWith("/") && !cleaned.endsWith("://")) {
      cleaned = cleaned.slice(0, -1);
    }

    // Remove paths and query parameters for suggestion generation
    if (cleaned.includes("/") && !cleaned.endsWith("://")) {
      const protocolMatch = cleaned.match(/^https?:\/\//);
      if (protocolMatch) {
        const afterProtocol = cleaned.substring(protocolMatch[0].length);
        const domain = afterProtocol.split("/")[0];
        cleaned = protocolMatch[0] + domain;
      } else {
        cleaned = cleaned.split("/")[0];
      }
    }

    // Remove query parameters
    if (cleaned.includes("?")) {
      cleaned = cleaned.split("?")[0];
    }

    // Remove hash fragments
    if (cleaned.includes("#")) {
      cleaned = cleaned.split("#")[0];
    }

    return cleaned;
  }

  normalizeURLString(input) {
    let normalized = input;

    // Remove default ports
    normalized = normalized.replace(/:80([/?#]|$)/, "$1");
    normalized = normalized.replace(/:443([/?#]|$)/, "$1");

    // Remove www. prefix for suggestions
    normalized = normalized.replace(/^(https?:\/\/)?www\./, "$1");

    return normalized;
  }

  normalizeURL() {
    const value = this.input.value.trim();
    if (!value) return;

    let normalized = value;

    // Add https:// if no protocol
    if (
      !normalized.startsWith("http://") &&
      !normalized.startsWith("https://")
    ) {
      normalized = `https://${normalized}`;
    }

    // Remove default ports
    normalized = normalized.replace(/:80([/?#]|$)/, "$1");
    normalized = normalized.replace(/:443([/?#]|$)/, "$1");

    // Only update if different
    if (normalized !== value) {
      this.input.value = normalized;
    }
  }

  updateDatalist(suggestions) {
    this.clearDatalist();

    suggestions.forEach((suggestion) => {
      const option = document.createElement("option");
      option.value = suggestion;
      this.datalist.appendChild(option);
    });
  }

  clearDatalist() {
    this.datalist.innerHTML = "";
  }
}

// Auto-initialize for any URL input with a datalist
document.addEventListener("DOMContentLoaded", function () {
  const urlInputs = document.querySelectorAll('input[type="url"][list]');
  urlInputs.forEach((input) => {
    const datalistId = input.getAttribute("list");
    const datalist = document.getElementById(datalistId);
    if (datalist) {
      new URLCompletion(input, datalist);
    }
  });
});
