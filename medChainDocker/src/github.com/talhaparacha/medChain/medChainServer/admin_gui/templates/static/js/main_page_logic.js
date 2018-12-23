var info = {};

function initInfo() {
  return {logged_in:false, role:"", public_key:"", user_id:"", name:"", super_admin_id:"", created:false, hospital_name:""};
}

function DoLogin(){
  var public_key_val = $("#public_key_input").val();
  info.public_key = public_key_val;
  GetSignerInfo();
}

function DisplaySignerInfo(){
  if(!info.logged_in){
    return
  }
  if(info.role == "super_admin"){
    $("#signer_role").text("Head of Hospital");
  }else if(info.role == "admin"){
    $("#signer_role").text("Administrator");
  }else if(info.role == "manager"){
    $("#signer_role").text("Manager");
  }else if(info.role == "user"){
    $("#signer_role").text("User");
  }else{
    $("#signer_role").text("Unknown");
  }
  $("#hospital_name").text(info.hospital_name);
  $("#signer_name").text(info.name);
  $("#signer_id").text(info.user_id);
  $("#login_div").hide();
  $("#display_div").show();
}


function GetSignerInfo(){
  var json_val = {"public_key": info.public_key};
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

function GetHospitalInfo(){
  var json_val = {"identity": info.super_admin_id};
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

function ShowLoginError(error) {
  alert("Error: "+ error);
  $("#login_error").text("Error: "+ error);
  $("#login_error").show();
  info = initInfo();
}

function UpdateUserInfo(data){
  info.role = data["role"];
  info.user_id = data["id"];
  info.name = data["name"];
  info.super_admin_id = data["super_admin_id"];
  info.created = data["is_created"];
  info.user_darc_base_id = data["darc_base_id"];
  GetHospitalInfo();
}

function UpdateHospitalInfo(data){
  info.hospital_name = data["hospital_name"];
  info.logged_in = true;
  DisplaySignerInfo();
}

$(document).ready(
  function(){
    info = initInfo();
    $("#login_button").click(DoLogin);
    $("#display_div").hide();
  }
);
