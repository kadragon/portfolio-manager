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

  function showToast(message, type) {
    var container = document.getElementById("toast-container");
    if (!container) {
      container = document.createElement("div");
      container.id = "toast-container";
      container.className = "toast toast-end toast-bottom z-50";
      document.body.appendChild(container);
    }

    var alertClass = type === "error" ? "alert-error" : "alert-success";
    var toast = document.createElement("div");
    toast.className = "alert " + alertClass + " text-sm py-2 px-4 shadow-lg";
    toast.setAttribute("role", "status");
    var span = document.createElement("span");
    span.textContent = message;
    toast.appendChild(span);
    container.appendChild(toast);

    setTimeout(function () {
      toast.style.opacity = "0";
      toast.style.transition = "opacity 300ms ease-out";
      setTimeout(function () {
        toast.remove();
        if (container.children.length === 0) {
          container.remove();
        }
      }, 300);
    }, 3000);
  }

  // Title for the next drawer open, captured from the triggering element on click
  // (htmx:afterSwap detail.elt is not reliably the trigger).
  var pendingDrawerTitle = "수정";

  function drawerEl() {
    return document.getElementById("drawer");
  }

  function openDrawer(title) {
    var d = drawerEl();
    if (!d) return;
    var panel = d.querySelector("aside");
    var heading = document.getElementById("drawer-title");
    if (heading && title) {
      heading.textContent = title;
    }
    d.classList.remove("invisible", "opacity-0");
    d.classList.add("opacity-100");
    d.setAttribute("aria-hidden", "false");
    document.body.style.overflow = "hidden";
    // next frame so the transform transition runs from the off-screen state
    requestAnimationFrame(function () {
      if (panel) panel.classList.remove("translate-x-full");
    });
    if (panel) {
      var field = panel.querySelector("input, select, textarea");
      if (field) field.focus();
    }
  }

  function closeDrawer() {
    var d = drawerEl();
    if (!d) return;
    var panel = d.querySelector("aside");
    if (panel) panel.classList.add("translate-x-full");
    d.classList.remove("opacity-100");
    d.classList.add("opacity-0");
    d.setAttribute("aria-hidden", "true");
    document.body.style.overflow = "";
    setTimeout(function () {
      d.classList.add("invisible");
      var body = document.getElementById("drawer-body");
      if (body) body.innerHTML = "";
    }, 200);
  }

  function drawerIsOpen() {
    var d = drawerEl();
    return d && d.getAttribute("aria-hidden") === "false";
  }

  document.addEventListener("DOMContentLoaded", function () {
    document.body.addEventListener("htmx:beforeRequest", function (event) {
      var message =
        event.detail && event.detail.elt && event.detail.elt.dataset.requestMessage
          ? event.detail.elt.dataset.requestMessage
          : "요청 중…";
      setStatus(message);
    });

    // A form loaded into the drawer slides the panel open once swapped in.
    document.body.addEventListener("htmx:afterSwap", function (event) {
      if (event.detail.target && event.detail.target.id === "drawer-body") {
        openDrawer(pendingDrawerTitle);
      }
    });

    document.body.addEventListener("htmx:afterRequest", function (event) {
      setStatus("");
      if (event.detail.successful) {
        showToast("요청이 완료되었습니다.", "success");
        // A successful submit from inside the drawer (save) closes the panel.
        // The edit GET that *opens* the drawer is triggered from outside it.
        var elt = event.detail.elt;
        if (elt && elt.closest && elt.closest("#drawer")) {
          closeDrawer();
        }
      }
    });

    // Close affordances: overlay, ✕ / 취소 buttons, and Escape.
    document.body.addEventListener("click", function (event) {
      if (event.target.closest("[data-drawer-close]")) {
        closeDrawer();
        return;
      }
      var opener = event.target.closest("[data-drawer-title]");
      if (opener) {
        pendingDrawerTitle = opener.getAttribute("data-drawer-title") || "수정";
      }
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && drawerIsOpen()) {
        closeDrawer();
      }
    });

    document.body.addEventListener("htmx:responseError", function () {
      setStatus("");
      showToast("요청에 실패했습니다. 다시 시도해 주세요.", "error");
    });

    document.body.addEventListener("htmx:sendError", function () {
      setStatus("");
      showToast("네트워크 오류가 발생했습니다.", "error");
    });
  });
})();
