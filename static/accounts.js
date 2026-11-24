// Progressive enhancement for the account pages: reveal, copy and live filter.
// Nothing here builds markup from data; text is only ever read or set through
// textContent and value.
"use strict";

(function () {
  function csrfToken() {
    var field = document.querySelector("input[name=_csrf]");
    return field ? field.value : "";
  }

  function fallbackCopy(text) {
    var area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.className = "offscreen";
    document.body.appendChild(area);
    area.select();
    var ok = false;
    try {
      ok = document.execCommand("copy");
    } catch (e) {
      ok = false;
    }
    document.body.removeChild(area);
    return ok ? Promise.resolve() : Promise.reject(new Error("copy failed"));
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text).catch(function () {
        return fallbackCopy(text);
      });
    }
    return fallbackCopy(text);
  }

  // flash shows a short message on a button, then puts its label back.
  function flash(button, message) {
    if (button.dataset.label === undefined) {
      button.dataset.label = button.textContent;
    }
    button.textContent = message;
    window.setTimeout(function () {
      button.textContent = button.dataset.label;
    }, 1500);
  }

  function copyWithFeedback(button, getText) {
    Promise.resolve()
      .then(getText)
      .then(copyText)
      .then(function () {
        flash(button, "Copied");
      })
      .catch(function () {
        flash(button, "Failed");
      });
  }

  function fetchPassphrase(url) {
    return fetch(url, {
      method: "POST",
      credentials: "same-origin",
      headers: { "X-CSRF-Token": csrfToken() },
    }).then(function (response) {
      // An expired session is answered with a redirect to the login page,
      // which fetch follows: that is HTML, not a passphrase. Send the reader
      // to sign in again instead of copying whatever came back.
      var json = (response.headers.get("Content-Type") || "").indexOf("application/json") === 0;
      if (response.status === 401 || response.redirected || !json) {
        sessionExpired();
        throw new Error("session expired");
      }
      if (!response.ok) {
        throw new Error("status " + response.status);
      }
      return response.json();
    }).then(function (body) {
      if (typeof body.passphrase !== "string") {
        throw new Error("unexpected response");
      }
      return body.passphrase;
    });
  }

  function sessionExpired() {
    var next = window.location.pathname + window.location.search;
    window.location.assign("/login?expired=1&next=" + encodeURIComponent(next));
  }

  // Buttons that need JavaScript start hidden.
  document.querySelectorAll("button[hidden]").forEach(function (button) {
    button.hidden = false;
  });

  document.addEventListener("click", function (event) {
    var button = event.target.closest("button");
    if (!button) {
      return;
    }
    if (button.dataset.reveal) {
      var input = document.getElementById(button.dataset.reveal);
      var show = input.type === "password";
      input.type = show ? "text" : "password";
      button.textContent = show ? "Hide" : "Show";
    } else if (button.dataset.copyInput) {
      var source = document.getElementById(button.dataset.copyInput);
      copyWithFeedback(button, function () {
        return source.value;
      });
    } else if (button.dataset.copyText !== undefined) {
      copyWithFeedback(button, function () {
        return button.dataset.copyText;
      });
    } else if (button.dataset.copyPassphrase !== undefined) {
      copyWithFeedback(button, function () {
        return fetchPassphrase(button.dataset.url);
      });
    }
  });

  // Live filter: hides the cards whose data-search lacks the query. The
  // server-side ?q= filter stays the fallback.
  var filter = document.querySelector("input[data-filter]");
  if (filter) {
    var grid = document.getElementById(filter.dataset.filter);
    // Enter submits to the server only when the box no longer holds the
    // query the page was filtered with, so the search can be widened.
    filter.form.addEventListener("submit", function (event) {
      if (filter.value.trim() === (filter.dataset.q || "")) {
        event.preventDefault();
      }
    });
    filter.addEventListener("input", function () {
      if (!grid) {
        return;
      }
      var query = filter.value.trim().toLowerCase();
      grid.querySelectorAll("[data-card]").forEach(function (card) {
        card.hidden = query !== "" && card.dataset.search.indexOf(query) === -1;
      });
    });
  }
})();
