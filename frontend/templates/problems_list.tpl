{{define "problems_list"}}
	{{range $i, $e := .}}
		<div class="panel panel-default">
		  <div class="panel-heading">
		    <h4 class="panel-title">
		      <a data-toggle="collapse" data-parent="#accordion" href="#collapse{{$i}}">{{$e.Title}}</a>
		    </h4>
		  </div>
		  <div id="collapse{{$i}}" class="panel-collapse collapse {{if eq $i 0}}in{{end}}">
		    <div class="panel-body">{{$e.ShortDesc}}</div>
		    <div>
		      <input type="button" class="newproblem" name="{{$e.Name}}" value="Attempt {{$e.Name}}"/>
		    </div>
		  </div>
		</div>
	{{end}}
{{end}}