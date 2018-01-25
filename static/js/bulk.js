document.addEventListener('DOMContentLoaded', function () {
  // Standard behavior for all bulk-action buttons
  var bulkButtons = document.querySelectorAll("button.bulk-action");
  for (var i = 0; i < bulkButtons.length; i++) {
    var btn = bulkButtons[i];
    btn.addEventListener("click", clickCallback(btn));
  }
})

function clickCallback(btn) {
  return function(e) {
    btn.setAttribute("disabled", "disabled");
    var postLocation = btn.dataset["action"];
    fetch(postLocation, {method: "POST", credentials: "same-origin"}).then(function(response) {
      if (response.status != 200) {
        btn.removeAttribute("disabled");
        alert("Error!  Please try again or contact support.");
        return;
      }

      var id = btn.dataset["toggleOnSuccess"];
      var el = document.getElementById(id);

      // On the bulk downloads page, we don't have a "Queue" button, so we
      // have to check for null elements
      if (el != null) {
        el.removeAttribute("disabled");
      }

      // bulk-downloads needs to hide the row
      if (btn.dataset["isRemove"] == "1") {
        var row = btn.closest(".bulk-row");
        if (row != null) {
          row.remove();
        }
      }
      return response.text();
    }).then(function(responseBody) {
      if (responseBody != null) {
        var queueInfo = document.getElementById("queue-info")
        if (queueInfo != null) {
          queueInfo.innerHTML = responseBody;
        }
      }
    });
  };
}
