

function DisplayHospitalList(){
  $('#hospital_list_table tbody').html("");
  for( var user_index in info.hospital_list){
    var user_info = info.hospital_list[user_index]
    var status = user_info.created ? "Approved" : "Not Approved";
    $('#hospital_list_table tbody').append('<tr><td>'+user_info.hospital_name+'</td><td>'+user_info.super_admin_id+'</td><td>'+user_info.super_admin_name+'</td><td>'+status+'</td></tr>');
  }
  $("#list_hospital_div").show();
  if(info.signer.role == "super_admin"){
    AllowAddHospital();
  }
}

function DisplayAdminList(){
  $('#admin_list_table tbody').html("");
  for( var user_index in info.hospital.admin_list){
    var user_info = info.hospital.admin_list[user_index]
    var status = user_info.created ? "Approved" : "Not Approved";
    $('#admin_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+user_info.id+'</td><td>'+status+'</td></tr>');
  }
  $("#list_admin_div").show();
  if(info.signer.role == "super_admin"){
    AllowAddAdmin();
  }
}

function DisplayManagerList(){
  $('#manager_list_table tbody').html("");
  $('#new_project_managers_table tbody').html("");
  for( var user_index in info.hospital.manager_list){
    var user_info = info.hospital.manager_list[user_index]
    var status = user_info.created ? "Approved" : "Not Approved";
    $('#manager_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+user_info.id+'</td><td>'+status+'</td></tr>');
    if(user_info.created){
        $('#new_project_managers_table tbody').append('<tr><td>'+user_info.name+'</td><td><input type="checkbox" name="new_project_managers" value="'+user_info.id+'"></td></tr>');
    }
  }
  $("#list_manager_div").show();
  if(info.signer.role == "admin"){
    AllowAddManager();
  }
}

function DisplayUserList(){
  $('#user_list_table tbody').html("");
  $('#new_project_authorizations_table tbody').html("");
  for( var user_index in info.hospital.user_list){
    var user_info = info.hospital.user_list[user_index];
    var status = user_info.created ? "Approved" : "Not Approved";
    $('#user_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+user_info.id+'</td><td>'+status+'</td></tr>');
    if(user_info.created){
      $('#new_project_authorizations_table tbody').append('<tr><td>'+user_info.name+'</td><td><input type="checkbox" name="new_project_aggregated_users" value="'+user_info.id+'"></td><td><input type="checkbox" name="new_project_obfuscated_users" value="'+user_info.id+'"></td></tr>');
    }
  }
  $("#list_user_div").show();
  if(info.signer.role == "admin"){
    AllowAddUser();
  }
}

function DisplayProjectList(){
  $('#project_list_table tbody').html("");
  for( var project_index in info.project_list){
    var project_info = info.project_list[project_index];
    var status = project_info.created ? "Approved" : "Not Approved";
    $('#project_list_table tbody').append('<tr><td>'+project_info.name+'</td><td>'+project_info.id+'</td><td>'+status+'</td></tr>');
  }
  $("#list_project_div").show();
  if(info.signer.role == "admin"){
    AllowAddProject();
  }
}

function DisplaySignerActionList(){
  $('#signer_action_list_table tbody').html("");
  for( var action_index in info.signer.actions){
    var action_info = info.signer.actions[action_index];
    var display_string = '<tr><td>'+action_info.action_id+'</td><td>'+action_info.action.action_type+'</td><td>'+action_info.status+'</td><td><ul>'
    for(var signer_id in action_info.signatures){
      var has_signed = action_info.signatures[signer_id]
      display_string += "<li>"+signer_id + " : "
      if(has_signed){
        display_string += "<span class='text-success'>Approved</span>"
      }else{
        display_string += "<span>Waiting</span>"
      }
      display_string +="</li>"
    }
    display_string += "</ul></td><td>"
    if(action_info.status == "Approved"){
      display_string += '<button class="btn btn-success">Commit</button>'
    }
    if(action_info.status == "Waiting" || action_info.status == "Approved"){
      display_string += '<button class="btn btn-danger">Cancel</button>'
    }
    display_string += "</td></tr>"
    $('#signer_action_list_table tbody').append(display_string);
  }
  $("#list_signer_action_div").show();
}

function DisplayWaitingActionList(){
  $('#waiting_action_list_table tbody').html("");
  for( var action_index in info.signer.waiting_actions){
    var action_info = info.signer.waiting_actions[action_index];
    var display_string = '<tr><td>'+action_info.action_id+'</td><td>'+action_info.action.action_type+'</td><td>'+action_info.initiator_id+'</td><td><button class="btn btn-success">Approve</button><button class="btn btn-danger">Deny</button></td></tr>'
    $('#waiting_action_list_table tbody').append(display_string);
  }
  $("#list_waiting_action_div").show();
}
