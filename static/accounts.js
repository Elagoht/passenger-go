// Passenger page behaviour: copying, revealing, generating, the live
// filter with keyboard navigation, and copy notifications.
// Text is only ever set through textContent.
(function () {
  "use strict";

  function csrfToken() {
    var field = document.querySelector("input[name=_csrf]");
    return field ? field.value : "";
  }

  // Toast ------------------------------------------------------------------

  var toastEl = document.querySelector("[data-toast]");
  var toastTimer = 0;

  function toast(message, isError) {
    if (!toastEl) {
      return;
    }
    toastEl.textContent = message;
    toastEl.classList.toggle("is-error", Boolean(isError));
    toastEl.hidden = false;
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.hidden = true;
    }, 2200);
  }

  // Clipboard ---------------------------------------------------------------

  function legacyCopy(text) {
    var area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.className = "visually-hidden";
    document.body.appendChild(area);
    area.select();
    var ok = false;
    try {
      ok = document.execCommand("copy");
    } catch (e) {
      ok = false;
    }
    document.body.removeChild(area);
    if (!ok) {
      throw new Error("copy failed");
    }
  }

  function copyText(text) {
    if (typeof text !== "string") {
      return Promise.reject(new Error("nothing to copy"));
    }
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text).catch(function () {
        legacyCopy(text);
      });
    }
    legacyCopy(text);
    return Promise.resolve();
  }

  // The label never changes, so the ticket does not shift; the toast says
  // what was copied and the button flashes green.
  function markDone(button) {
    if (!button) {
      return;
    }
    button.classList.add("is-done");
    window.setTimeout(function () {
      button.classList.remove("is-done");
    }, 1200);
  }

  function copyWithFeedback(button, what, getText) {
    Promise.resolve()
      .then(getText)
      .then(copyText)
      .then(function () {
        markDone(button);
        toast(what + " copied");
      })
      .catch(function (error) {
        if (error && error.message === "session expired") {
          return;
        }
        toast("Could not copy. Try again.", true);
      });
  }

  function sessionExpired() {
    var here = window.location.pathname + window.location.search;
    window.location.assign(here === "/" ? "/login" : "/login?next=" + encodeURIComponent(here));
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

  function copyPassword(button) {
    var name = button.dataset.name || "Password";
    copyWithFeedback(button, name + " password", function () {
      return fetchPassphrase(button.dataset.url);
    });
  }

  function copyUsername(button) {
    var name = button.dataset.name || "Username";
    copyWithFeedback(button, name + " username", function () {
      return button.dataset.copyText;
    });
  }

  // Generator and strength (mirrors internal/generator and internal/strength)

  var LOWERS = "abcdefghijklmnopqrstuvyz";
  var UPPERS = "ABCDEFGHIJKLMNOPQRSTUVYZ";
  var NUMBERS = "0123456789";
  var SPECIALS = "!@#$%^&*()_+-=[]{}|;:,.<>?/";
  var CHARS = LOWERS + UPPERS + NUMBERS + SPECIALS;

  function randomIndex(n) {
    // Rejection sampling keeps the choice uniform.
    var limit = Math.floor(4294967296 / n) * n;
    var buf = new Uint32Array(1);
    do {
      window.crypto.getRandomValues(buf);
    } while (buf[0] >= limit);
    return buf[0] % n;
  }

  function generate(length) {
    var out = [];
    var i;
    for (i = 0; i < length; i++) {
      out.push(CHARS[randomIndex(CHARS.length)]);
    }
    var positions = [];
    for (i = 0; i < length; i++) {
      positions.push(i);
    }
    for (i = positions.length - 1; i > 0; i--) {
      var j = randomIndex(i + 1);
      var t = positions[i];
      positions[i] = positions[j];
      positions[j] = t;
    }
    var sets = [LOWERS, UPPERS, NUMBERS, SPECIALS];
    for (i = 0; i < 8; i++) {
      var set = sets[i % 4];
      out[positions[i]] = set[randomIndex(set.length)];
    }
    return out.join("");
  }

  function strengthLabel(value) {
    var length = Array.from(value).length;
    var score;
    if (length <= 5) {
      score = -2;
    } else if (length <= 7) {
      score = 0;
    } else if (length <= 12) {
      score = 1;
    } else if (length <= 16) {
      score = 2;
    } else if (length <= 20) {
      score = 3;
    } else {
      score = 4;
    }
    [/[a-z]/, /[A-Z]/, /[0-9]/, /[!-/:-@[-`{-~]/].forEach(function (re) {
      if (re.test(value)) {
        score += 1;
      }
    });
    if (score <= 2) {
      return "weak";
    }
    return score <= 5 ? "fair" : "strong";
  }

  function updateMeter(input) {
    var meter = document.querySelector('[data-meter-for="' + input.id + '"]');
    if (!meter) {
      return;
    }
    meter.textContent = input.value ? "Strength: " + strengthLabel(input.value) : "";
  }

  // Wiring -----------------------------------------------------------------

  // Controls that need JavaScript start hidden.
  document.querySelectorAll("button[hidden]").forEach(function (button) {
    button.hidden = false;
  });

  // Mark the current section in the top navigation.
  document.querySelectorAll(".topnav > a").forEach(function (link) {
    var path = window.location.pathname;
    var href = link.getAttribute("href");
    if (href === path || (href === "/" && path.indexOf("/accounts/") === 0)) {
      link.setAttribute("aria-current", "page");
    }
  });

  // Close the "More" menu when clicking elsewhere or pressing Escape.
  var menu = document.querySelector("details.menu");
  if (menu) {
    document.addEventListener("click", function (event) {
      if (menu.open && !menu.contains(event.target)) {
        menu.open = false;
      }
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && menu.open) {
        menu.open = false;
        menu.querySelector("summary").focus();
      }
    });
  }

  document.querySelectorAll("[data-meter-for]").forEach(function (meter) {
    var input = document.getElementById(meter.dataset.meterFor);
    if (input) {
      updateMeter(input);
      input.addEventListener("input", function () {
        updateMeter(input);
      });
    }
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
      copyWithFeedback(button, button.dataset.name || "Text", function () {
        return "value" in source ? source.value : source.textContent.trim();
      });
    } else if (button.dataset.copyText !== undefined) {
      copyUsername(button);
    } else if (button.dataset.copyPassphrase !== undefined) {
      copyPassword(button);
    } else if (button.dataset.generateInto) {
      var target = document.getElementById(button.dataset.generateInto);
      target.value = generate(20);
      target.type = "text";
      var reveal = document.querySelector('[data-reveal="' + target.id + '"]');
      if (reveal) {
        reveal.textContent = "Hide";
      }
      // Lets the session banner know the form has unsaved changes.
      target.dispatchEvent(new Event("input", { bubbles: true }));
      toast("New password generated");
    }
  });

  // Finder: live filter and keyboard navigation over the tickets ------------

  var filter = document.querySelector("input[data-filter]");
  if (!filter) {
    return;
  }
  var list = document.getElementById(filter.dataset.filter);
  var tickets = list ? Array.prototype.slice.call(list.querySelectorAll("[data-card]")) : [];
  var selected = null;

  // Keyboard shortcuts and auto-focus only where there is a keyboard, so a
  // phone does not pop its on-screen keyboard on every visit.
  var hasKeyboard = window.matchMedia("(hover: hover) and (pointer: fine)").matches;
  var hint = document.querySelector("[data-keys-hint]");
  if (hint && hasKeyboard) {
    hint.hidden = false;
  }
  if (hasKeyboard && filter.hasAttribute("data-autofocus")) {
    filter.focus();
  }

  function visibleTickets() {
    return tickets.filter(function (ticket) {
      return !ticket.hidden;
    });
  }

  function select(ticket) {
    if (selected) {
      selected.classList.remove("is-selected");
    }
    selected = ticket || null;
    if (selected) {
      selected.classList.add("is-selected");
      selected.scrollIntoView({ block: "nearest" });
    }
  }

  function move(step) {
    var visible = visibleTickets();
    if (!visible.length) {
      return;
    }
    var index = visible.indexOf(selected);
    index = index === -1 ? (step > 0 ? 0 : visible.length - 1) : Math.min(Math.max(index + step, 0), visible.length - 1);
    select(visible[index]);
  }

  function applyFilter() {
    var query = filter.value.trim().toLowerCase();
    tickets.forEach(function (ticket) {
      ticket.hidden = query !== "" && ticket.dataset.search.indexOf(query) === -1;
    });
    // While searching, the first match is ready for Enter.
    select(query ? visibleTickets()[0] : null);
  }

  filter.addEventListener("input", applyFilter);

  filter.form.addEventListener("submit", function (event) {
    var serverQuery = filter.dataset.q || "";
    // A page filtered on the server can be widened with a new search.
    if (serverQuery !== "" && filter.value.trim() !== serverQuery) {
      return;
    }
    event.preventDefault();
  });

  function copyFromSelected(username) {
    if (!selected) {
      return;
    }
    var button = selected.querySelector(username ? "[data-copy-text]" : "[data-copy-passphrase]");
    if (button) {
      button.click();
    }
  }

  // Keys work anywhere on the page, not only in the search box: clicking a
  // ticket or a copy button must not break arrow navigation.
  document.addEventListener("keydown", function (event) {
    if (event.metaKey || event.ctrlKey || event.altKey || event.isComposing) {
      return;
    }
    var target = event.target;
    var tag = (target.tagName || "").toLowerCase();
    var inFilter = target === filter;
    var typingElsewhere = !inFilter && (tag === "input" || tag === "textarea" || tag === "select" || target.isContentEditable);
    if (typingElsewhere) {
      return;
    }

    if (event.key === "/" && !inFilter) {
      event.preventDefault();
      filter.focus();
      filter.select();
    } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      move(event.key === "ArrowDown" ? 1 : -1);
    } else if (event.key === "Enter" && selected && (inFilter || tag === "body" || target.matches("[data-card]"))) {
      // On a focused button or link, Enter keeps its own meaning.
      event.preventDefault();
      copyFromSelected(event.shiftKey);
    } else if (event.key === "Escape") {
      if (inFilter && filter.value) {
        filter.value = "";
        applyFilter();
      } else {
        select(null);
      }
    }
  });

  // Using a ticket with the mouse makes it the selected one, so the arrows
  // continue from there.
  if (list) {
    list.addEventListener("click", function (event) {
      var ticket = event.target.closest("[data-card]");
      if (ticket) {
        if (selected) {
          selected.classList.remove("is-selected");
        }
        selected = ticket;
        selected.classList.add("is-selected");
      }
    });
  }

  // A page the server already filtered (which also matches notes) keeps its
  // results; the first one is ready for Enter.
  if (filter.value.trim()) {
    select(visibleTickets()[0]);
  }
})();
