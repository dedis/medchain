function GetGeneralInfo(){
  $.ajax
    ({
        type: "GET",
        url: '/info',
        success: UpdateGeneralInfo,
        failure: ShowLoginError,
    });
}

function UpdateGeneralInfo(data){
  info.all_super_admins_id = data["all_super_admins_darc_base_id"]
}

function GetSignerInfo(){
  var json_val = {"public_key": info.signer.public_key};
  $.ajax
    ({
        type: "POST",
        url: '/info/user',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateUserInfo,
        error: ShowLoginError,
        contentType: 'application/json'
    });
}

function UpdateUserInfo(data){
  info.signer.role = data["role"];
  info.signer.id = data["id"];
  info.signer.name = data["name"];
  info.hospital.id = data["super_admin_id"];
  info.signer.created = data["is_created"];
  info.signer.darc_base_id = data["darc_base_id"];
  info.logged_in = true;
  window.sessionStorage.setItem("MedChainLoggedIn", info.logged_in);
  window.sessionStorage.setItem("MedChainPublicKey", info.signer.public_key);
  GetHospitalInfo();
  RefreshAllInfo();
  DisplaySignerInfo();
}

function RefreshAllInfo(){
  if(info.signer.created){
    switch (info.signer.role) {
      case "super_admin":
        GetSignerActions();
        GetWaitingActions();
        GetHospitalList();
        GetAdminList();
        break;
      case "admin":
        GetSignerActions();
        GetWaitingActions();
        GetManagerList();
        GetUserList();
        GetProjectList();
        break;
      case "manager":
        GetSignerActions();
        GetWaitingActions();
        GetUserList();
        GetProjectList();
        break;
      case "user":
        GetProjectList();
    }
  }
}


function DisplaySignerInfo(){
  if(!info.logged_in){
    return
  }

  switch (info.signer.role) {
    case "super_admin":
      $("#signer_role").text("Head of Hospital");
      break;
    case "admin":
      $("#signer_role").text("Administrator");
      break;
    case "manager":
      $("#signer_role").text("Manager");
      break;
    case "user":
      $("#signer_role").text("User");
      break;
    default:
      $("#signer_role").text("Unknown");
  }

  $("#signer_name").text(info.signer.name);
  $("#signer_id").text(info.signer.id);
  if(info.signer.created){
    $("#signer_status").html("<span class='text-success'>Approved</span>");
  }else{
    $("#signer_status").html("<span class='text-warning'>Not Approved</span>");
  }
  $("#signer_info_div").show();
  $("#login_error").text("");
}

function GetHospitalInfo(){
  var json_val = {"identity": info.hospital.id};
  $.ajax
    ({
        type: "POST",
        url: '/info/hospital',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateHospitalInfo,
        failure: ShowLoginError,
        contentType: 'application/json'
    });
}

function UpdateHospitalInfo(data){
  info.hospital.name = data["hospital_name"];
  info.hospital.super_admin_name = data["super_admin_name"];
  info.hospital.admin_list_id = data["admin_list_darc_base_id"];
  info.hospital.manager_list_id = data["manager_list_darc_base_id"];
  info.hospital.user_list_id = data["user_list_darc_base_id"];
  info.hospital.created = data["is_created"];
  DisplayHospitalInfo();
}

function DisplayHospitalInfo(){
  $("#hospital_name").text(info.hospital.name);
  $("#super_admin_name").text(info.hospital.super_admin_name);
  if(info.hospital.created){
    $("#hospital_status").html("<span class='text-success'>Approved</span>");
  }else{
    $("#hospital_status").html("<span class='text-warning'>Not Approved</span>");
  }
  $("#hospital_info_div").show();
}


function ShowLoginError(error) {
  $("#login_error").text("Error: "+ error["responseText"]);
  resetInfo();
  showLogin();
}

function GetUserList(){
  getGenericUserList("user", UpdateUserList);
}

function getGenericUserList(role, callback){
  var json_val = {"super_admin_id": info.hospital.id, "role":role};
  $.ajax
    ({
        type: "POST",
        url: '/list/users',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: callback,
        failure: ShowLoginError,
        contentType: 'application/json'
    });
}

function GetManagerList(){
  getGenericUserList("manager", UpdateManagerList);
}

function GetAdminList(){
  getGenericUserList("admin", UpdateAdminList);
}

function GetHospitalList(){
  $.ajax
    ({
        type: "GET",
        url: '/list/hospitals',
        success: UpdateHospitalList,
        failure: ShowLoginError,
    });
}



function GetProjectList(){
  var json_val = {"id": info.signer.id};
  $.ajax
    ({
        type: "POST",
        url: '/list/projects',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateProjectList,
        failure: ShowLoginError,
        contentType: 'application/json'
    });
}

function UpdateHospitalList(data){
  var hospital_array = data["hospitals"];
  info.hospital_list = [];
  for (var i in hospital_array) {
    var hospital_info = hospital_array[i];
    info.hospital_list.push( {hospital_name:hospital_info["hospital_name"], super_admin_id:hospital_info["super_admin_id"], super_admin_name:hospital_info["super_admin_name"], created:hospital_info["is_created"]} );
  }
  DisplayHospitalList();
}

function UpdateUserList(data){
  var user_array = data["users"];
  info.hospital.user_list = [];
  for (var i in user_array) {
    var user_info = user_array[i];
    info.hospital.user_list.push({name:user_info["name"], id:user_info["id"], created:user_info["is_created"]});
  }
  DisplayUserList();
}

function UpdateManagerList(data){
  var user_array = data["users"];
  info.hospital.manager_list = [];
  for (var i in user_array) {
    var user_info = user_array[i];
    info.hospital.manager_list.push({name:user_info["name"], id:user_info["id"], created:user_info["is_created"]});
  }
  DisplayManagerList();
}

function UpdateAdminList(data){
  var user_array = data["users"];
  info.hospital.admin_list = [];
  for (var i in user_array) {
    var user_info = user_array[i];
    info.hospital.admin_list.push({name:user_info["name"], id:user_info["id"], created:user_info["is_created"]});
  }
  DisplayAdminList();
}

function UpdateProjectList(data){
  var project_array = data["projects"];
  info.project_list = [];
  for (var i in project_array) {
    var project_info = project_array[i];
    info.project_list.push({name:project_info["name"], id:project_info["id"], created:project_info["is_created"]});
  }
  DisplayProjectList();
}

function GetSignerActions(){
  var json_val = {"id": info.signer.id};
  $.ajax
    ({
        type: "POST",
        url: '/list/actions',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateSignerActionList,
        failure: ShowActionError,
        contentType: 'application/json'
    });
}

function UpdateSignerActionList(data){
  info.signer.actions = data.actions;
  DisplaySignerActionList();
}

function GetWaitingActions(){
  var json_val = {"id": info.signer.id};
  $.ajax
    ({
        type: "POST",
        url: '/list/actions/waiting',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateWaitingActionList,
        failure: ShowActionError,
        contentType: 'application/json'
    });
}

function UpdateWaitingActionList(data){
  info.signer.waiting_actions = data.actions;
  DisplayWaitingActionList();
}
