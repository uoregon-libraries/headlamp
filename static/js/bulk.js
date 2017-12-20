document.addEventListener('DOMContentLoaded', function () {
  var bulkButtons = document.querySelectorAll("button.bulk-action");
  bulkButtons.forEach(function(btn) {
    btn.addEventListener("click", function(e) {
      btn.setAttribute("disabled", "disabled");
      var postLocation = btn.dataset["action"];
      fetch(postLocation, {method: "POST", credentials: "same-origin"}).then(function(response) {
        console.log(response);
        if (response.status == 200) {
          var id = btn.dataset["toggleOnSuccess"];
          console.log("ID is " + id);
          var el = document.getElementById(id);
          console.log(el);
          el.removeAttribute("disabled");
        }
        else {
          btn.removeAttribute("disabled");
          alert("Error!  Please try again or contact support.");
        }
      });
    });
  });
});
