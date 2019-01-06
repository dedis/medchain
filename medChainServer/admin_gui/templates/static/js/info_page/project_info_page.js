function GetProjectInfo(name){
  var json_val = {"name":name};
  $.ajax
    ({
        type: "POST",
        url: '/info/project',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: DisplayProjectInfo,
        failure:DisplayProjectError,
        error:DisplayProjectError,
        contentType: 'application/json'
    });
}

function DisplayProjectInfo(data){
  $("#project_info_name").text(data.name);
  if(data.is_created){
    $("#project_info_status").html("<span class='text-success'>Approved</span>");
  }else{
    $("#project_info_status").html("<span class='text-warning'>Not Approved</span>");
  }
  $('#project_info_managers tbody').html("");
  for( var index in data.managers){
    var manager_info = data.managers[index];
    $('#project_info_managers tbody').append('<tr><td>'+manager_info.name +'</td><td><button class="btn btn-sm btn-info" id="button_show_project_manager_id_'+manager_info.id+'" onclick="GetUserInfo(\''+manager_info.id+'\');">Info</button></td></tr>');
  }
  $('#project_info_authorizations tbody').html("");
  for( var index in data.users){
    var user_info = data.users[index];
    var row_string = '<tr><td>'+user_info.name+'</td><td><button class="btn btn-sm btn-info" id="button_show_project_user_id_'+user_info.id+'" onclick="GetUserInfo(\''+user_info.id+'\');">Info</button></td>'
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
    $('#project_info_authorizations tbody').append(row_string);
  }
  $("#project_info_dialog").dialog("open");
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

function DisplayProjectError(){
  $("#project_info_dialog").html("<h1>Failed loading the information</h1>");
  $("#project_info_dialog").dialog("open");
}
