{{define "main"}}
<!DOCTYPE html>
<html>
  <head>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="/static/vendor/bootstrap/css/bootstrap.min.css">
    <script src="/static/vendor/jquery/jquery.min.js"></script>
    <script src="/static/vendor/bootstrap/js/bootstrap.min.js"></script>
    <script src="/static/app.js"> </script>
  </head>
  <body>
    <div class="container">
      <h2>Problems list</h2>
      <p><strong>Note:</strong> This a work in progress web application.  For details visit http://madhavjha.com.</p>
      <div class="panel-group" id="accordion">
      	{{template "problems_list" .}}
      </div>
      <div id="parent">
        <a id="child"></a>
      </div>
    </div>
  </body>
</html>
{{end}}