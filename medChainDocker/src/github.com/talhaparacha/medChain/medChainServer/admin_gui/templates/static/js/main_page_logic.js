var info = {};

function initInfo() {
  return {logged_in:false, all_super_admins_id:"", signer:{role:"", public_key:"", id:"", name:"", created:false, darc_base_id:"", projects:[]},   hospital_list:[], hospital:{id:"", name:"", created:false, super_admin_name:"",admin_list_id:"", manager_list_id:"", user_list_id:"", user_list:[], manager_list:[], admin_list:[]}};
}

function DoLogin(){
  var public_key_val = $("#public_key_input").val();
  info.signer.public_key = public_key_val;
  GetGeneralInfo();
  GetSignerInfo();
  $("#display_div").show();
  $("#login_div").hide();
}

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
  GetHospitalInfo();
  switch (info.signer.role) {
    case "super_admin":
      GetHospitalList();
      GetAdminList();
      break;
    case "admin":
      GetManagerList();
      GetUserList();
      break;
    case "manager":
      GetUserList();
      break;
    case "user":
      // GetProjectList();
  }
  DisplaySignerInfo();
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
    $("#signer_status").text("Approved");
  }else{
    $("#signer_status").text("Not Approved");
  }
  $("#signer_info_div").show();
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
    $("#hospital_status").text("Approved");
  }else{
    $("#hospital_status").text("Not Approved");
  }
  $("#hospital_info_div").show();
}


function ShowLoginError(error) {
  $("#login_error").text("Error: "+ error["responseText"]);
  $("#login_div").show();
  showLogin();
  info = initInfo();
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
  var json_val = {"identity": info.signer.id};
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
  alert(JSON.stringify(data));
}

function DisplayUserList(){
  $('#user_list_table tbody').html("");
  for( var user_index in info.hospital.user_list){
    var user_info = info.hospital.user_list[user_index];
    var status = user_info.created ? "Approved" : "Not Approved";
    $('#user_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+user_info.id+'</td><td>'+status+'</td></tr>');
  }
  $("#list_user_div").show();
  if(info.signer.role == "admin"){
    AllowAddUser();
  }
}

function AllowAddUser(){
  $("#add_user_div").show();
}

function AddUser(){
  var public_key = $("#new_user_public_key_input").val();
  var name = $("#new_user_name_input").val();
  var json_val = {"name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
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
  GetUserList();
}

function DisplayManagerList(){
  $('#manager_list_table tbody').html("");
  for( var user_index in info.hospital.manager_list){
    var user_info = info.hospital.manager_list[user_index]
    var status = user_info.created ? "Approved" : "Not Approved";
    $('#manager_list_table tbody').append('<tr><td>'+user_info.name+'</td><td>'+user_info.id+'</td><td>'+status+'</td></tr>');
  }
  $("#list_manager_div").show();
  if(info.signer.role == "admin"){
    AllowAddManager();
  }
}

function AllowAddManager(){
  $("#add_manager_div").show();
}

function AddManager(){
  var public_key = $("#new_manager_public_key_input").val();
  var name = $("#new_manager_name_input").val();
  var json_val = {"name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
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
  GetManagerList();
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

function AllowAddAdmin(){
  $("#add_admin_div").show();
}

function AddAdmin(){
  var public_key = $("#new_admin_public_key_input").val();
  var name = $("#new_admin_name_input").val();
  var json_val = {"name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
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
  GetAdminList();
}



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

function AllowAddHospital(){
  $("#add_hospital_div").show();
}

function AddHospital(){
  var public_key = $("#new_hospital_public_key_input").val();
  var hospital_name = $("#new_hospital_name_input").val();
  var super_admin_name = $("#new_hospital_head_name_input").val();
  var json_val = {"hospital_name":hospital_name, "super_admin_name":super_admin_name, "new_public_key":public_key}
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
  GetHospitalList();
}

function showLogin(){
  $("#login_div").show();
  $("#display_div").hide();
  $("#signer_info_div").hide();
  $("#hospital_info_div").hide();
  $("#list_hospital_div").hide();
  $("#add_hospital_div").hide();
  $("#list_admin_div").hide();
  $("#add_admin_div").hide();
  $("#list_manager_div").hide();
  $("#add_manager_div").hide();
  $("#list_user_div").hide();
  $("#add_user_div").hide();
  $("#list_project_div").hide();
}


$(document).ready(
  function(){
    info = initInfo();
    $("#login_button").click(DoLogin);
    $("#new_hospital_button").click(AddHospital);
    $("#new_admin_button").click(AddAdmin);
    $("#new_manager_button").click(AddManager);
    $("#new_user_button").click(AddUser);
    showLogin();
  }
);
