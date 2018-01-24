document.addEventListener('DOMContentLoaded', function () {
  // Standard behavior for all bulk-action buttons
  var bulkButtons = document.querySelectorAll("button.bulk-action");
  for (var i = 0; i < bulkButtons.length; i++) {
    var btn = bulkButtons[i];
    btn.addEventListener("click", clickCallback(btn));
  }

  // Special behavior for the "remove" buttons on the bulk download page
  var bulkRows = document.querySelectorAll(".bulk-row");
  for (var i = 0; i < bulkRows.length; i++) {
    var row = bulkRows[i];
    var btn = row.querySelector("button.bulk-action[data-is-remove='1']");
    btn.addEventListener("click", removeRowCallback(row));
  }
})

function clickCallback(btn) {
  return function(e) {
    btn.setAttribute("disabled", "disabled");
    var postLocation = btn.dataset["action"];
    fetch(postLocation, {method: "POST", credentials: "same-origin"}).then(function(response) {
      if (response.status == 200) {
        var id = btn.dataset["toggleOnSuccess"];
        var el = document.getElementById(id);
        // On the bulk downloads page, we don't have a "Queue" button, so we
        // have to check for null elements
        if (el != null) {
          el.removeAttribute("disabled");
        }
      }
      else {
        btn.removeAttribute("disabled");
        alert("Error!  Please try again or contact support.");
      }
    });
  };
}

function removeRowCallback(row) {
  return function(e) {
    row.remove();
  };
}
