// Session expiry warning: a minute before the session ends, ask the server for
// the real end (another tab may have extended it) and, if it is still near,
// show the banner with a countdown. At the end, go to the login page.
// The server and the browser may disagree about the time of day, so the page
// never compares clocks: it takes the time left from the server and counts it
// down on the browser's monotonic clock (performance.now).
// Text is only ever set through textContent.
"use strict";

(function () {
  var WARN_MS = 60000;
  var banner = document.getElementById("session-banner");
  var RETRY_MS = 15000;
  var remaining = parseInt(document.body.dataset.sessionRemaining, 10);
  if (!banner || !remaining || remaining < 0) {
    return;
  }
  var deadline = performance.now() + remaining;

  function left() {
    return deadline - performance.now();
  }

  var message = banner.querySelector("[data-session-message]");
  var errorText = banner.querySelector("[data-session-error]");
  var extendButton = banner.querySelector("[data-session-extend]");
  var signOutButton = banner.querySelector("[data-session-signout]");
  var logoutForm = document.getElementById("logout-form");
  var checkTimer = 0;
  var tickTimer = 0;
  var dirty = false;

  function csrfToken() {
    var field = document.querySelector("input[name=_csrf]");
    return field ? field.value : "";
  }

  function stop() {
    window.clearTimeout(checkTimer);
    window.clearInterval(tickTimer);
    checkTimer = 0;
    tickTimer = 0;
  }

  function hide() {
    banner.hidden = true;
  }

  function leave() {
    stop();
    var here = window.location.pathname + window.location.search;
    window.location.assign(here === "/" ? "/login" : "/login?next=" + encodeURIComponent(here));
  }

  function render() {
    var seconds = Math.max(0, Math.ceil(left() / 1000));
    var text = "Your session ends in " + seconds + (seconds === 1 ? " second." : " seconds.");
    if (dirty) {
      text += " You have unsaved changes.";
    }
    message.textContent = text;
  }

  // schedule arms the check for one minute before the end (at once when that
  // moment has passed).
  function schedule(delay) {
    stop();
    if (delay === undefined) {
      delay = Math.max(0, left() - WARN_MS);
    }
    checkTimer = window.setTimeout(check, delay);
  }

  function tick() {
    if (left() <= 0) {
      finish();
      return;
    }
    render();
  }

  function warn() {
    render();
    banner.hidden = false;
    if (!tickTimer) {
      tickTimer = window.setInterval(tick, 1000);
    }
  }

  // apply takes the server's answer; the end may have moved.
  function apply(body) {
    deadline = performance.now() + body.remainingMs;
    errorText.textContent = "";
    if (left() > WARN_MS) {
      hide();
      schedule();
    } else {
      stop();
      warn();
    }
  }

  function ask(url, options) {
    options.credentials = "same-origin";
    return fetch(url, options).then(function (response) {
      if (response.status === 401) {
        return null;
      }
      if (!response.ok) {
        throw new Error("status " + response.status);
      }
      return response.json();
    });
  }

  // check asks the server for the real end. When the server cannot be reached,
  // the end this page knows is all there is: with more than a minute left
  // there is nothing to show yet, so try again shortly.
  function check() {
    ask("/session", { method: "GET" })
      .then(function (body) {
        if (body === null) {
          leave();
        } else {
          apply(body);
        }
      })
      .catch(function () {
        if (left() <= 0) {
          leave();
        } else if (left() > WARN_MS) {
          hide();
          schedule(RETRY_MS);
        } else {
          stop();
          warn();
        }
      });
  }

  // finish runs when the known end has come: another tab may have extended
  // the session meanwhile, so ask once more before leaving.
  function finish() {
    stop();
    ask("/session", { method: "GET" })
      .then(function (body) {
        if (body === null || body.remainingMs <= 0) {
          leave();
        } else {
          apply(body);
        }
      })
      .catch(leave);
  }

  extendButton.addEventListener("click", function () {
    extendButton.disabled = true;
    ask("/session/extend", { method: "POST", headers: { "X-CSRF-Token": csrfToken() } })
      .then(function (body) {
        if (body === null) {
          leave();
        } else {
          apply(body);
        }
      })
      .catch(function () {
        errorText.textContent = "Could not extend the session. Try again.";
      })
      .then(function () {
        extendButton.disabled = false;
      });
  });

  signOutButton.addEventListener("click", function () {
    if (logoutForm) {
      logoutForm.submit();
    }
  });

  function markDirty() {
    dirty = true;
    if (!banner.hidden) {
      render();
    }
  }
  document.querySelectorAll("form[data-track]").forEach(function (form) {
    form.addEventListener("input", markDirty);
    form.addEventListener("change", markDirty);
  });

  // A sleeping laptop stops timers and drops the network; look again as soon
  // as the tab is back, the window is focused or the connection returns.
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") {
      check();
    }
  });
  window.addEventListener("focus", check);
  window.addEventListener("online", check);

  schedule();
})();
