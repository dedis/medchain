var info = {};

function initInfo() {
  return {logged_in:false, all_super_admins_id:"", signer:{role:"", public_key:"", id:"", name:"", created:false, darc_base_id:"", actions:[], waiting_actions:[]},   project_list:[], hospital_list:[], hospital:{id:"", name:"", created:false, super_admin_name:"",admin_list_id:"", manager_list_id:"", user_list_id:"", user_list:[], manager_list:[], admin_list:[]}};
}

function DoLogin(){
  var public_key_val = $("#public_key_input").val();
  info.signer.public_key = public_key_val;
  GetGeneralInfo();
  GetSignerInfo();
  $("#display_div").show();
  $("#login_div").hide();
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
  $("#add_project_div").hide();
  $("#list_signer_action_div").hide();
  $("#list_waiting_action_div").hide();
}

$(document).ready(
  function(){
    info = initInfo();
    $("#login_button").click(DoLogin);
    $("#new_hospital_button").click(AddHospital);
    $("#new_admin_button").click(AddAdmin);
    $("#new_manager_button").click(AddManager);
    $("#new_user_button").click(AddUser);
    $("#new_project_button").click(AddProject);
    showLogin();
  }
);
