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

  document.addEventListener("DOMContentLoaded", function () {
    document.body.addEventListener("htmx:beforeRequest", function (event) {
      var message =
        event.detail && event.detail.elt && event.detail.elt.dataset.requestMessage
          ? event.detail.elt.dataset.requestMessage
          : "요청 중…";
      setStatus(message);
    });

    document.body.addEventListener("htmx:afterRequest", function (event) {
      setStatus("");
      if (event.detail.successful) {
        showToast("요청이 완료되었습니다.", "success");
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
