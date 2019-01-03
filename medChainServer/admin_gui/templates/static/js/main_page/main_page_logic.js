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

function OpenNewHospitalDialog(){
  $("#add_hospital_div").dialog("open");
}

function OpenNewAdminDialog(){
  $("#add_admin_div").dialog("open");
}

function OpenNewManagerDialog(){
  $("#add_manager_div").dialog("open");
}
function OpenNewUserDialog(){
  $("#add_user_div").dialog("open");
}

function OpenNewProjectDialog(){
  $("#add_project_div").dialog("open");
}


function showLogin(){
  $("#login_div").show();
  $("#display_div").hide();
  $("#signer_info_div").hide();
  $("#hospital_info_div").hide();
  $("#list_hospital_div").hide();
  $("#open_new_hospital_dialog_button").hide();;
  $("#add_hospital_div").dialog({
    title:"Add a new Hospital",
    modal:true,
    autoOpen: false
  });
  $("#list_admin_div").hide();
  $("#open_new_admin_dialog_button").hide();
  $("#add_admin_div").dialog({
    title:"Add a new Administrator",
    modal:true,
    autoOpen: false
  })
  $("#list_manager_div").hide();
  $("#open_new_manager_dialog_button").hide();
  $("#add_manager_div").dialog({
    title:"Add a new Manager",
    modal:true,
    autoOpen: false
  })
  $("#list_user_div").hide();
  $("#open_new_user_dialog_button").hide();
  $("#add_user_div").dialog({
    title:"Add a new User",
    modal:true,
    autoOpen: false
  })
  $("#list_project_div").hide();
  $("#open_new_project_dialog_button").hide();
  $("#add_project_div").dialog({
    title:"Add a new Project",
    minWidth:600,
    modal:true,
    autoOpen: false
  })
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
    $("#open_new_hospital_dialog_button").click(OpenNewHospitalDialog);
    $("#open_new_admin_dialog_button").click(OpenNewAdminDialog);
    $("#open_new_manager_dialog_button").click(OpenNewManagerDialog);
    $("#open_new_user_dialog_button").click(OpenNewUserDialog);
    $("#open_new_project_dialog_button").click(OpenNewProjectDialog);
    showLogin();
  }
);
