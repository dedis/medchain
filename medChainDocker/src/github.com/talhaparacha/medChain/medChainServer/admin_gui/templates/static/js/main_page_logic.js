var info = {};

function initInfo() {
  return {logged_in:false, role:"", public_key:"", user_id:"", name:"", super_admin_id:"", created:false};
}

function DoLogin(){
  var public_key_val = $("#public_key_input").val();
  info.public_key = public_key_val;
  GetSignerRole();
}

function DisplayInfo(){
  alert(info.role);
  if(!info.logged_in){
    return
  }
  if(info.role == "hospital"){
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
  $("#signer_name").text(info.name);
  $("#signer_id").text(info.user_id);
  $("#login_div").hide();
  $("#display_div").show();
}

function GetSignerRole(){
  var json_val = {"public_key": info.public_key};
  $.ajax
    ({
        type: "POST",
        url: '/info/type',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateSignerRole,
        failure: ShowLoginError,
        contentType: 'application/json'
    });
}

function GetUserInfo(){
  if(info.role == ""){
    return
  }
  var json_val = {"public_key": info.public_key};
  $.ajax
    ({
        type: "POST",
        url: '/info/'+info.role,
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: UpdateUserInfo,
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

function UpdateSignerRole(data){
  info.role = data["type"];
  GetUserInfo();
}

function UpdateUserInfo(data){
  if(info.role != "hospital"){
    info.user_id = data["id"];
    info.name = data["name"];
  }else{
    info.user_id = data["super_admin_id"];
    info.name = data["hospital_name"];
  }
  info.super_admin_id = data["super_admin_id"];
  info.created = data["is_created"];
  info.user_darc_base_id = data["darc_base_id"];
  info.logged_in = true;
  DisplayInfo();
}

$(document).ready(
  function(){
    info = initInfo();
    $("#login_button").click(DoLogin);
    $("#display_div").hide();
  }
);
