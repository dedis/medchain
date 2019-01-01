function AllowAddUser(){
  $("#add_user_div").show();
}

function AddUser(){
  var public_key = $("#new_user_public_key_input").val();
  var name = $("#new_user_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
  $.ajax
    ({
        type: "POST",
        url: '/add/user',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddUserCallback,
        failure:
        function(error){
          $("#add_user_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
}

function AddUserCallback(data){
  AddAction(data);
  GetUserList();
}



function AllowAddManager(){
  $("#add_manager_div").show();
}

function AddManager(){
  var public_key = $("#new_manager_public_key_input").val();
  var name = $("#new_manager_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
  $.ajax
    ({
        type: "POST",
        url: '/add/manager',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddManagerCallback,
        failure:
        function(error){
          $("#add_manager_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
}

function AddManagerCallback(data){
  AddAction(data);
  GetManagerList();
}


function AllowAddAdmin(){
  $("#add_admin_div").show();
}

function AddAdmin(){
  var public_key = $("#new_admin_public_key_input").val();
  var name = $("#new_admin_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
  $.ajax
    ({
        type: "POST",
        url: '/add/admin',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddAdminCallback,
        failure:
        function(error){
          $("#add_admin_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
}

function AddAdminCallback(data){
  AddAction(data);
  GetAdminList();
}



function AllowAddHospital(){
  $("#add_hospital_div").show();
}

function AddHospital(){
  var public_key = $("#new_hospital_public_key_input").val();
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
        failure:
        function(error){
          $("#add_hospital_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
}

function AddHospitalCallback(data){
  AddAction(data);
  GetHospitalList();
}

function AllowAddProject(){
  $("#add_project_div").show();
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
        failure:
        function(error){
          $("#add_project_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
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
  var client_url = $("#local_signer_url_input").val();
  var private_key = $("#private_key_input").val();
  var json_val = {"action_info":action_info, "public_key":info.signer.public_key, "private_key":private_key}
  $.ajax
    ({
        type: "POST",
        url: client_url + '/sign',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: ApproveActionStatus,
        failure: ShowActionError,
        contentType: 'application/json'
    });
}

function ApproveActionStatus(data){
  alert(JSON.stringify(data));
  $.ajax
    ({
        type: "POST",
        url: '/approve/action',
        dataType: 'json',
        data: JSON.stringify(data),
        success: function(){
          GetSignerActions();
          GetWaitingActions();
        },
        failure: ShowActionError,
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
  $.ajax
    ({
        type: "POST",
        url: '/commit/action',
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

function CancelAction(action_info){
  alert("Cancel "+action_info.action_id);
}
