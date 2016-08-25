function showStartCodingLink(data) {
  var el = document.createElement('a');
  el.id = "child";
  el.href = "cui/"+ data.ticket_id;
  el.target = "_blank";
  el.appendChild(document.createTextNode("Click to solve " + data.problem_id));
  var parent = document.getElementById('parent');
  var child = document.getElementById('child');
  parent.replaceChild(el, child);

}

function createNewTicket(problemId) {
    $.ajax({
      url: "/cui/new",
      data: {'problem_id': problemId},
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
    el.addEventListener('click', function(){
      createNewTicket(el.name);
    });
  });
});




