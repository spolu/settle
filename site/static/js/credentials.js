new Clipboard('.copy');

var getBangParameter = function getBangParameter(sParam) {
  var idx = window.location.hash.indexOf('?')
  if (idx == -1) {
    return null;
  }
  var sPageURL = decodeURIComponent(window.location.hash.substr(idx + 1));
  var sURLVariables = sPageURL.split('&');
  var sParameterName;
  var i;

  for (var i = 0; i < sURLVariables.length; i++) {
    sParameterName = sURLVariables[i].split('=');

    if (sParameterName[0] === sParam) {
      return sParameterName[1] === undefined ? true : sParameterName[1];
    }
  }
};

function rollUser() {
  var fqn = (env == "qa") ? "qa-register" : "register";
  var url = "https://" + fqn + ".settle.network/users/" + username +"/roll";

  var c = confirm("You are about to roll your password, your old password won't be usable aymore. Continue?")
  if (!c) {
    return
  }

  $.ajax({
    url: url,
    type: 'POST',
    dataType: "json",
    data: { secret: secret },
    success: function(data) {
      $("#credentials #address .value").text(data.credentials.address)
      $("#credentials #password .value").text(data.credentials.password)
    },
    error: function(xhr, status, error) {
      var err = "Unexpected Error: please contact register@settle.network with the URL of the page.";
      try {
        var body = JSON.parse(xhr.responseText)
        if (body['error']) {
          err = "Error: " + body['error']['message']
        }
      }
      finally {
        $("#credentials .error").text(err)
      }
    }
  });
}


var secret = "";
var username = "";
var env = ""

$(document).ready(function() {
  secret = getBangParameter("secret")
  username = getBangParameter("username")

  env = getBangParameter("env")
  if (env != "qa") {
    env = "prod"
  } else {
    // quite an hack
    $("#wrapper pre code").text("settle -env=qa login")
  }

  console.log("Retrieving credentials:"+
              " env="+env+" username="+username+" secret="+ secret)

  var fqn = (env == "qa") ? "qa-register" : "register";
  var url = "https://" + fqn + ".settle.network/users/" + username +
    "?secret=" + secret;

  $.ajax({
    url: url,
    dataType: "json",
    success: function(data) {
      $("#credentials #address .value").text(data.credentials.address)
      $("#credentials #password .value").text(data.credentials.password)
    },
    error: function(xhr, status, error) {
      var err = "Unexpected Error: please contact register@settle.network with the URL of the page.";
      try {
        var body = JSON.parse(xhr.responseText)
        if (body['error']) {
          err = "Error: " + body['error']['message']
        }
      }
      finally {
        $("#credentials .error").text(err)
      }
    }
  });
})
