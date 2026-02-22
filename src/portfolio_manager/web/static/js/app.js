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

  document.addEventListener("DOMContentLoaded", function () {
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
