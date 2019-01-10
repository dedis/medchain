function AllowAddUser(){
  $("#open_new_user_dialog_button").show();
}

function AddUser(public_key){
  var name = $("#new_user_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
  $.ajax
    ({
        type: "POST",
        url: '/add/user',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddUserCallback,
        failure:ShowNewUserError,
        error:ShowNewUserError,
        contentType: 'application/json'
    });
}

function ShowNewUserError(error) {
  $("#new_user_error").text("Error: "+ error["responseText"]);
}

function AddUserCallback(data){
  $("#add_user_div :input").each(function(){$(this).val("");});
  $("#new_user_error").text("");
  $("#add_user_div").dialog("close");
  AddAction(data);
  GetUserList();
}



function AllowAddManager(){
  $("#open_new_manager_dialog_button").show();
}

function AddManager(public_key){
  var name = $("#new_manager_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
  $.ajax
    ({
        type: "POST",
        url: '/add/manager',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddManagerCallback,
        failure:ShowNewManagerError,
        error:ShowNewManagerError,
        contentType: 'application/json'
    });
}

function ShowNewManagerError(error) {
  $("#new_manager_error").text("Error: "+ error["responseText"]);
}

function AddManagerCallback(data){
  $("#add_manager_div :input").each(function(){$(this).val("");});
  $("#new_manager_error").text("");
  $("#add_manager_div").dialog("close");
  AddAction(data);
  GetManagerList();
}


function AllowAddAdmin(){
  $("#open_new_admin_dialog_button").show();
}

function AddAdmin(public_key){
  var name = $("#new_admin_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
  $.ajax
    ({
        type: "POST",
        url: '/add/admin',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddAdminCallback,
        failure:ShowNewAdminError,
        error:ShowNewAdminError,
        contentType: 'application/json'
    });
}

function ShowNewAdminError(error) {
  $("#new_admin_error").text("Error: "+ error["responseText"]);
}

function AddAdminCallback(data){
  $("#add_admin_div :input").each(function(){$(this).val("");});
  $("#new_admin_error").text("");
  $("#add_admin_div").dialog("close");
  AddAction(data);
  GetAdminList();
}



function AllowAddHospital(){
  $("#open_new_hospital_dialog_button").show();
}

function AddHospital(public_key){
  var hospital_name = $("#new_hospital_name_input").val();
  var super_admin_name = $("#new_hospital_head_name_input").val();
  var json_val = {"initiator":info.signer.id, "hospital_name":hospital_name, "super_admin_name":super_admin_name, "new_public_key":public_key}
  $.ajax
    ({
        type: "POST",
        url: '/add/hospital',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddHospitalCallback,
        failure:ShowNewHospitalError,
        error:ShowNewHospitalError,
        contentType: 'application/json'
    });
}

function ShowNewHospitalError(error) {
  $("#new_hospital_error").text("Error: "+ error["responseText"]);
}

function AddHospitalCallback(data){
  $("#add_hospital_div :input").each(function(){$(this).val("");});
  $("#new_hospital_error").text("");
  $("#add_hospital_div").dialog("close");
  AddAction(data);
  GetHospitalList();
}

function AllowAddProject(){
  $("#open_new_project_dialog_button").show();
}

function AddProject(){
  var project_name = $("#new_project_name_input").val();
  var managers = GetNewProjectManagers();
  var queries = GetNewProjectQueryMapping();
  var json_val = {"initiator":info.signer.id, "name":project_name, "managers":managers, "queries":queries}
  $.ajax
    ({
        type: "POST",
        url: '/add/project',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddProjectCallback,
        failure:ShowNewProjectError,
        error:ShowNewProjectError,
        contentType: 'application/json'
    });
}

function ShowNewProjectError(error) {
  $("#new_project_error").text("Error: "+ error["responseText"]);
}

function GetNewProjectManagers(){
  var result = [];
  $('input[type="checkbox"][name="new_project_managers"]:checked').each(function(){
    result.push($(this).val());
  });
  return result
}

function GetNewProjectQueryMapping(){
  var AggregatedQueryUsers = [];
  $('input[type="checkbox"][name="new_project_aggregated_users"]:checked').each(function(){
    AggregatedQueryUsers.push($(this).val());
  });
  var ObfuscatedQueryUsers = [];
  $('input[type="checkbox"][name="new_project_obfuscated_users"]:checked').each(function(){
    ObfuscatedQueryUsers.push($(this).val());
  });
  return {"AggregatedQuery":AggregatedQueryUsers, "ObfuscatedQuery":ObfuscatedQueryUsers}
}

function AddProjectCallback(data){
  $("#add_project_div :input[type='text']").each(function(){$(this).val("");});
  $("#add_project_div :input[type='checkbox']").each(function(){$(this).prop('checked', false);});
  $("#new_project_error").text("");
  $("#add_project_div").dialog("close");
  AddAction(data);
  GetProjectList();
}

function AddAction(data_val){
  var json_val = {"action": data_val}
  $.ajax
    ({
        type: "POST",
        url: '/add/action',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success:function(){
          GetSignerActions();
          GetWaitingActions();
        },
        failure:ShowActionError,
        contentType: 'application/json'
    });
}

function ApproveAction(action_info){
  let action_info_string = JSON.stringify(action_info);
  $("#approve_action_and_sign_button").click(function(){
    let action_info_string_copy = action_info_string;
    let action_info_copy = JSON.parse(action_info_string_copy);
    SignAndApproveAction(action_info_copy);
  });
  $("#signature_information_dialog").dialog("open");
}


function SignAndApproveAction(action_info){
  var client_url = $("#local_signer_url_input").val();
  var action_info_string = JSON.stringify(action_info);
  var callback_success = function(private_key){
    var action_info_string_copy = action_info_string;
    var action_info_copy = JSON.parse(action_info_string_copy);
    var json_val = {"action_info":action_info_copy, "public_key":info.signer.public_key, "private_key":private_key}
    $.ajax
      ({
          type: "POST",
          url: client_url + '/sign',
          dataType: 'json',
          data: JSON.stringify(json_val),
          success: ApproveActionStatus,
          failure: function(error){
            $("#approve_action_error").text("Error: "+ error["responseText"]);
          },
          contentType: 'application/json'
      });
  }
  var callback_error = function(error){
    $("#approve_action_error").text("Error: "+ error["responseText"]);
  }
  handleKeyFileSelect("private_key_input", callback_success, callback_error);
}

function ApproveActionStatus(data){
  $.ajax
    ({
        type: "POST",
        url: '/approve/action',
        dataType: 'json',
        data: JSON.stringify(data),
        success: function(){
          $("#signature_information_dialog").dialog("close");
          $("#approve_action_error").text("");
          GetSignerActions();
          GetWaitingActions();
        },
        failure: function(error){
          $("#approve_action_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
}

function DenyAction(action_info){
  var json_val = {"action_id": action_info.action_id, "signer_id": info.signer.id}
  $.ajax
    ({
        type: "POST",
        url: '/deny/action',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success:function(){
          GetSignerActions();
          GetWaitingActions();
        },
        failure:ShowActionError,
        contentType: 'application/json'
    });
}

function CommitAction(action_info){
  var json_val = {"transaction": action_info.action.transaction, "action_type": action_info.action.action_type}
  var action_id = action_info.action_id;
  $.ajax
    ({
        type: "POST",
        url: '/commit/action',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success:function(){
          let id = action_id;
          ChangeActionStatusToDone(id);
        },
        failure:ShowActionError,
        error:ShowActionError,
        contentType: 'application/json'
    });
}

function CancelAction(action_info){
  var json_val = {"transaction": action_info.action.transaction, "action_type": action_info.action.action_type}
  var action_id = action_info.action_id;
  $.ajax
    ({
        type: "POST",
        url: '/cancel/action',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success:function(){
          let id =action_id;
          ChangeActionStatusToCancelled(id);
        },
        failure:ShowActionError,
        error:ShowActionError,
        contentType: 'application/json'
    });
}

function ChangeActionStatusToDone(id){
  var json_val = {"action_id": id, "signer_id": info.signer.id}
  $.ajax
    ({
        type: "POST",
        url: '/update/action/done',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success:RefreshAllInfo,
        failure:ShowActionError,
        error:ShowActionError,
        contentType: 'application/json'
    });
}

function ChangeActionStatusToCancelled(id){
  var json_val = {"action_id": id, "signer_id": info.signer.id};
  $.ajax
    ({
        type: "POST",
        url: '/update/action/cancel',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success:RefreshAllInfo,
        failure:ShowActionError,
        error:ShowActionError,
        contentType: 'application/json'
    });
}

function ShowActionError(error) {
  alert("Error while submitting"+ error["responseText"]);
}
