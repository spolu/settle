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


$(document).ready(function() {
  var secret = getBangParameter("secret")
  var username = getBangParameter("username")

  var env = getBangParameter("env")
  if (env != "qa") {
    env = "prod"
  } else {
    // quite an hack
    $("#wrapper pre code").text("$> settle -env=qa login")
  }

  console.log("Retrieving credentials:"+
              " env="+env+" username="+username+" secret="+ secret)

  var protocol = (env == "qa") ? "http" : "https";
  var fqn = (env == "qa") ? "qa-register" : "register";
  var url = protocol + "://" + fqn + ".settle.network/users/" + username
  console.log(url)

  $.ajax({
    url: url,
    dataType: "json",
    success: function(data) {
      console.log(data)
    }
  });
})
