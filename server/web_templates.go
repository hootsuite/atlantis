package server

import (
	"html/template"
)

var indexTemplate = template.Must(template.New("index.html.tmpl").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>atlantis</title>
  <meta name="description" content="">
  <meta name="author" content="">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="/static/css/normalize.css">
  <link rel="stylesheet" href="/static/css/skeleton.css">
  <link rel="stylesheet" href="/static/css/custom.css">
  <link rel="icon" type="image/png" href="/static/images/atlantis-icon.png">
</head>
<body>
<div class="container">
  <section class="header">
    <a title="atlantis" href="/"><img src="/static/images/atlantis-icon.png"/></a>
    <p style="font-family: monospace, monospace; font-size: 1.1em; text-align: center;">atlantis</p>
  </section>
  <nav class="navbar">
    <div class="container">
    </div>
  </nav>
  <div class="navbar-spacer"></div>
  <br>
  <section>
    <p style="font-family: monospace, monospace; font-size: 1.0em; text-align: center;"><strong>Environments</strong></p>
    {{ if . }}
    {{ range . }}
      <a href="/detail?id={{.LockId}}">
        <div class="twelve columns button content lock-row">
        <div class="list-title">{{.RepoFullName}} - <span class="heading-font-size">#{{.PullNum}}</span></div>
        <div class="list-status"><code>Locked</code></div>
        <div class="list-timestamp"><span class="heading-font-size">{{.Time}}</span></div>
        </div>
      </a>
    {{ end }}
    {{ else }}
    <p class="placeholder">No environments found.</p>
    {{ end }}
  </section>
</div>
</body>
</html>
`))

var detailTemplate = template.Must(template.New("detail.html.tmpl").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>atlantis</title>
  <meta name="description" content="">
  <meta name="author" content="">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="/static/css/normalize.css">
  <link rel="stylesheet" href="/static/css/skeleton.css">
  <link rel="stylesheet" href="/static/css/custom.css">
  <link rel="icon" type="image/png" href="/static/images/atlantis-icon.png">
</head>
<body>
  <div class="container">
    <section class="header">
    <a title="atlantis" href="/"><img src="/static/images/atlantis-icon.png"/></a>
    <p style="font-family: monospace, monospace; font-size: 1.1em; text-align: center;">atlantis</p>
    <p style="font-family: monospace, monospace; font-size: 1.0em; text-align: center;"><strong>{{.LockKey}}</strong> <code>Locked</code></p>
    </section>
    <div class="navbar-spacer"></div>
    <br>
    <section>
      <div class="eight columns">
        <h6><code>Repo Owner</code>: <strong>{{.RepoOwner}}</strong></h6>
        <h6><code>Repo Name</code>: <strong>{{.RepoName}}</strong></h6>
        <h6><code>Pull Request Link</code>: <a href="{{.PullRequestLink}}" target="_blank"><strong>{{.PullRequestLink}}</strong></a></h6>
        <h6><code>Locked By</code>: <strong>{{.LockedBy}}</strong></h6>
        <h6><code>Environment</code>: <strong>{{.Environment}}</strong></h6>
        <br>
      </div>
      <div class="four columns">
        <a class="button button-default" id="discardPlanUnlock">Discard Plan & Unlock</a>
      </div>
    </section>
  </div>
  <div id="discardMessageModal" class="modal">
    <!-- Modal content -->
    <div class="modal-content">
      <div class="modal-header">
        <span class="close">&times;</span>
      </div>
      <div class="modal-body">
        <p><strong>Are you sure you want to discard the plan and unlock?</strong></p>
        <input class="button-primary" id="discardYes" type="submit" value="Yes">
        <input type="button" class="cancel" value="Cancel">
      </div>
    </div>
  </div>
<script>
  // Get the modal
  var modal = document.getElementById('discardMessageModal');

  // Get the button that opens the modal
  var btn = document.getElementById("discardPlanUnlock");
  var btnDiscard = document.getElementById("discardYes");

  // Get the <span> element that closes the modal
  var span = document.getElementsByClassName("close")[0];
  var cancelBtn = document.getElementsByClassName("cancel")[0];

  // When the user clicks the button, open the modal 
  btn.onclick = function() {
      modal.style.display = "block";
  }

  // When the user clicks on <span> (x), close the modal
  span.onclick = function() {
      modal.style.display = "none";
  }
  cancelBtn.onclick = function() {
    modal.style.display = "none";
  }

  btnDiscard.onclick = function() {
    console.log("plan discarded and unlocked!");
  }

  // When the user clicks anywhere outside of the modal, close it
  window.onclick = function(event) {
      if (event.target == modal) {
          modal.style.display = "none";
      }
  }
</script>
</body>
</html>
`))
