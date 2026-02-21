(function () {
  function setStatus(message) {
    var status = document.getElementById("app-request-status");
    var live = document.getElementById("app-live-region");
    if (status) {
      status.textContent = message;
    }
    if (live) {
      live.textContent = message;
    }
  }

  function initializeAutoRefreshToggle() {
    var toggle = document.getElementById("auto-refresh-toggle");
    var poller = document.getElementById("auto-refresh-poller");
    if (!toggle || !poller) {
      return;
    }

    var storageKey = "portfolio_manager_auto_refresh";
    var saved = null;
    try {
      saved = window.localStorage.getItem(storageKey);
    } catch (error) {
      saved = null;
    }

    applyState(saved !== "off");

    toggle.addEventListener("click", function () {
      var enabled = toggle.getAttribute("data-enabled") !== "true";
      applyState(enabled);
      try {
        window.localStorage.setItem(storageKey, enabled ? "on" : "off");
      } catch (error) {
        // Ignore localStorage failures.
      }
      setStatus(enabled ? "자동 새로고침을 켰습니다." : "자동 새로고침을 껐습니다.");
    });

    document.body.addEventListener("htmx:beforeRequest", function (event) {
      if (event.target === poller && poller.getAttribute("data-enabled") !== "true") {
        event.preventDefault();
      }
    });

    function applyState(enabled) {
      toggle.setAttribute("data-enabled", enabled ? "true" : "false");
      toggle.setAttribute("aria-pressed", enabled ? "true" : "false");
      toggle.textContent = enabled ? "자동 새로고침: 켜짐" : "자동 새로고침: 꺼짐";
      poller.setAttribute("data-enabled", enabled ? "true" : "false");

      if (enabled) {
        poller.setAttribute("hx-trigger", "every 30s");
      } else {
        poller.removeAttribute("hx-trigger");
      }

      if (window.htmx) {
        window.htmx.process(poller);
      }
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    initializeAutoRefreshToggle();

    document.body.addEventListener("htmx:beforeRequest", function (event) {
      var message =
        event.detail && event.detail.elt && event.detail.elt.dataset.requestMessage
          ? event.detail.elt.dataset.requestMessage
          : "요청 중…";
      setStatus(message);
    });

    document.body.addEventListener("htmx:afterRequest", function () {
      setStatus("요청이 완료되었습니다.");
    });

    document.body.addEventListener("htmx:responseError", function () {
      setStatus("요청에 실패했습니다. 다시 시도해 주세요.");
    });
  });
})();
