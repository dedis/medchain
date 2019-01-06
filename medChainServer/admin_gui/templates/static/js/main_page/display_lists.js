

function DisplayHospitalList(){
  $('#hospital_list_table tbody').html("");
  for( var user_index in info.hospital_list){
    var user_info = info.hospital_list[user_index]
    var status = user_info.created ? "<span class='text-success'>Approved</span>" : "<span class='text-warning'>Not Approved</span>";
    $('#hospital_list_table tbody').append('<tr><td>'+user_info.hospital_name+'</td><td>'+status+'</td><td><button class="btn btn-sm btn-info" id="button_show_hospital_id_'+user_info.super_admin_id+'" onclick="GetHospitalToDisplayInfo(\''+user_info.super_admin_id+'\');">See Details</button></td></tr>');
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
    var status = user_info.created ? "<span class='text-success'>Approved</span>" : "<span class='text-warning'>Not Approved</span>";
    $('#admin_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+status+'</td><td><button class="btn btn-sm btn-info" id="button_show_admin_id_'+user_info.id+'" onclick="GetUserInfo(\''+user_info.id+'\');">See Details</button></td></tr>');
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
    var status = user_info.created ? "<span class='text-success'>Approved</span>" : "<span class='text-warning'>Not Approved</span>";
    $('#manager_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+status+'</td><td><button class="btn btn-sm btn-info" id="button_show_manager_id_'+user_info.id+'" onclick="GetUserInfo(\''+user_info.id+'\');">See Details</button></td></tr>');
    if(user_info.created){
        $('#new_project_managers_table tbody').append('<tr><td>'+user_info.name+'  <button class="btn btn-sm btn-info" id="button_show_new_project_manager_id_'+user_info.id+'" onclick="GetUserInfo(\''+user_info.id+'\');">Info</button></td><td><input id="new_proj_manager_'+user_info.id+'" type="checkbox" name="new_project_managers" value="'+user_info.id+'"></td></tr>');
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
    var status = user_info.created ? "<span class='text-success'>Approved</span>" : "<span class='text-warning'>Not Approved</span>";
    $('#user_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+status+'</td><td><button class="btn btn-sm btn-info" id="button_show_user_id_'+user_info.id+'" onclick="GetUserInfo(\''+user_info.id+'\');">See Details</button></td></tr>');
    if(user_info.created){
      $('#new_project_authorizations_table tbody').append('<tr><td>'+user_info.name+'  <button class="btn btn-sm btn-info" id="button_show_new_project_user_id_'+user_info.id+'" onclick="GetUserInfo(\''+user_info.id+'\');">Info</button></td><td><input type="checkbox" id="agg_user_+'+user_info.id+'" name="new_project_aggregated_users" value="'+user_info.id+'"></td><td><input type="checkbox" id="obf_user_+'+user_info.id+'" name="new_project_obfuscated_users" value="'+user_info.id+'"></td></tr>');
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
    var status = project_info.created ? "<span class='text-success'>Approved</span>" : "<span class='text-warning'>Not Approved</span>";
    $('#project_list_table tbody').append('<tr><td>'+project_info.name+'</td><td>'+status+'</td><td><button class="btn btn-sm btn-info" id="button_show_project_name_'+project_info.name+'">See Details</button></td></tr>');
    let name = project_info.name;
    $("#button_show_project_name_"+project_info.name).click(
      function(){
        let name_copy = name;
        GetProjectInfo(name_copy);
      }
    );
  }
  $("#list_project_div").show();
  if(info.signer.role == "admin"){
    AllowAddProject();
  }
}

function DisplaySignerActionList(){
  $('#signer_action_list_table tbody').html("");
  info.signer.actions.reverse();
  for( var action_index in info.signer.actions){
    var action_info = info.signer.actions[action_index];
    var status = "";
    switch (action_info.status) {
      case "Approved":
        status = "<span class='text-success'>Approved</span>";
        break;
      case "Done":
        status = "<span class='text-success'>Done</span>";
        break;
      case "Denied":
        status = "<span class='text-danger'>Denied</span>";
        break;
      case "Cancelled":
        status = "<span class='text-danger'>Cancelled</span>";
        break;
      case "Waiting":
        status = "Waiting";
        break;
      default:
        status = "Unknown";
        break;
    }
    var display_string = '<tr><td>'+action_info.action_id+'</td><td>'+action_info.action.action_type+'</td><td>'+status+'</td><td><ul>'
    for(var signer_id in action_info.signatures){
      var has_signed = action_info.signatures[signer_id]
      display_string += '<li><button class="btn btn-sm btn-info" id="button_show_signer_id_'+signer_id+'" onclick="GetUserInfo(\''+signer_id+'\');">Signer Info</button>  :  '
      if(has_signed == "Approved"){
        display_string += "<span class='text-success'>Approved</span>"
      }else if (has_signed == "Waiting"){
        display_string += "<span>Waiting</span>"
      }else if (has_signed == "Denied"){
        display_string += "<span class='text-danger'>Denied</span>"
      }else if (has_signed == "NA") {
        display_string += "<span>Unknown</span>"
      }
      display_string +="</li>"
    }
    display_string += "</ul></td><td>"
    if(action_info.status == "Approved"){
      display_string += '<button id="commit_button_action_id_'+action_info.action_id+'" class="btn btn-success">Commit</button>'
    }
    if(action_info.status == "Waiting" || action_info.status == "Approved"){
      display_string += '<button id="cancel_button_action_id_'+action_info.action_id+'" class="btn btn-danger">Cancel</button>'
    }
    display_string += "</td></tr>"
    $('#signer_action_list_table tbody').append(display_string);

    let action_info_string = JSON.stringify(action_info);
    $('#commit_button_action_id_'+action_info.action_id).click(function(){
      let action_info_string_copy = action_info_string;
      let action_info_copy = JSON.parse(action_info_string_copy);
      CommitAction(action_info_copy);
    });

    $('#cancel_button_action_id_'+action_info.action_id).click(function(){
      let action_info_string_copy = action_info_string;
      let action_info_copy = JSON.parse(action_info_string_copy);
      alert(action_info_copy.action_id);
      CancelAction(action_info_copy);
    });

  }
  if(info.signer.actions.length > 0){
    $("#list_signer_action_div").show();
  }else{
    $("#list_signer_action_div").hide();
  }
}

function DisplayWaitingActionList(){
  $('#waiting_action_list_table tbody').html("");
  for( var action_index in info.signer.waiting_actions){
    let action_info = info.signer.waiting_actions[action_index];
    var display_string = '<tr><td>'+action_info.action_id+'</td><td>'+action_info.action.action_type+'</td><td>'+action_info.initiator_id+'</td><td><button class="btn btn-success" id="approve_button_action_id_'+action_info.action_id+'">Approve</button><button class="btn btn-danger" id="deny_button_action_id_'+action_info.action_id+'">Deny</button></td></tr>'
    $('#waiting_action_list_table tbody').append(display_string);
    let action_info_string = JSON.stringify(action_info);
    $('#approve_button_action_id_'+action_info.action_id).click(function(){
      let action_info_string_copy = action_info_string;
      let action_info_copy = JSON.parse(action_info_string_copy);
      ApproveAction(action_info_copy);
    });
    $('#deny_button_action_id_'+action_info.action_id).click(function(){
      let action_info_string_copy = action_info_string;
      let action_info_copy = JSON.parse(action_info_string_copy);
      DenyAction(action_info_copy);
    });
  }
  if(info.signer.waiting_actions.length > 0){
    $("#list_waiting_action_div").show();
  }else{
    $("#list_waiting_action_div").hide();
  }
}
