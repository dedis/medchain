function GetProjectInfo(){
  var curr_url_string = window.location.href;
  var curr_url = new URL(curr_url_string);
  var name = curr_url.searchParams.get("name");
  var json_val = {"name":name};
  $.ajax
    ({
        type: "POST",
        url: '/info/project',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: DisplayProjectInfo,
        failure:DisplayError,
        error:DisplayError,
        contentType: 'application/json'
    });
}

function DisplayProjectInfo(data){
  $("#name").text(data.name);
  if(data.is_created){
    $("#status").text("Approved");
    $("#darc").html("<a target='_blank' href='/gui/darc?base_id="+data.darc_base_id+"'>See details</a>");
  }else{
    $("#status").text("Not Approved");
    $("#darc").text("Not Created Yet");
  }
  for( var index in data.managers){
    var manager_info = data.managers[index];
    $('#managers').append("<li><a target='_blank' href='/gui/user?id="+manager_info.id+"'>"+manager_info.name+"</a></li>");
  }
  for( var index in data.users){
    var user_info = data.users[index];
    var row_string = '<tr><td><a target="_blank" href="/gui/user?id='+user_info.id+'">'+user_info.name+'</a></td>'
    if( idBelongsToList(user_info.id, data.queries["AggregatedQuery"]) ){
      row_string += "<td>Yes</td>";
    }else{
      row_string += "<td>No</td>";
    }
    if( idBelongsToList(user_info.id, data.queries["ObfuscatedQuery"]) ){
      row_string += "<td>Yes</td></tr>";
    }else{
      row_string += "<td>No</td></tr>";
    }
    $('#authorizations tbody').append(row_string);
  }
}

function idBelongsToList(id, list){
  for(var index in list){
    var elem = list[index];
    if(elem.id == id){
      return true
    }
  }
  return false
}

function DisplayError(){
  $("body").html("<h1>Failed loading the information</h1>")
}


$(document).ready(
  function(){
    GetProjectInfo();
  }
);
