function showStartCodingLink(data) {
  var el = document.createElement('a');
  el.href = "cui/"+ data.ticket_id;
  el.target = "_blank";
  el.appendChild(document.createTextNode("Click to solve " + data.problem_id));
  var parent = document.getElementById('dest');
  for(i=0; i < parent.childNodes.length; i++) {
    parent.removeChild(parent.childNodes[i]);
  }
  parent.appendChild(el);
}

function createNewTicket(problemId) {
    $.ajax({
      url: "/cui/new/" + problemId,
      beforeSend: function(xhr) {
      },
      error: function(err) {
        // error handler
        console.log(JSON.stringify(err));
      },
      success: function(data) {
        // success handler
        console.log(JSON.stringify(data));
        showStartCodingLink(data);
      }
    });
  }

$(document).ready(function() {
  $('.newproblem').each(function(i, el){
    console.log(el.name);
    el.addEventListener('click', function(){
      createNewTicket(el.name);
    });
  });
});




