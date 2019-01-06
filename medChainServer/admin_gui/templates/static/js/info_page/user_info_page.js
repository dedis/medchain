function GetUserInfo(identity){
  var json_val = {"identity":identity};
  $.ajax
    ({
        type: "POST",
        url: '/info/user',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: DisplayUserInfo,
        failure:DisplayUserError,
        error:DisplayUserError,
        contentType: 'application/json'
    });
}

function DisplayUserInfo(data){
  $("#user_info_name").text(data.name);
  $("#user_info_id").text(data.id);
  switch (data.role) {
    case "super_admin":
      $("#user_info_role").text("Head of Hospital");
      break;
    case "admin":
      $("#user_info_role").text("Administrator");
      break;
    case "manager":
      $("#user_info_role").text("Manager");
      break;
    case "user":
      $("#user_info_role").text("User");
      break;
    default:
      $("#user_info_role").text("Unknown");
  }
  if(data.is_created){
    $("#user_info_status").html("<span class='text-success'>Approved</span>");
  }else{
    $("#user_info_status").html("<span class='text-warning'>Not Approved</span>");
  }
  $("#user_info_hospital").html(data.hospital_name+'  <button class="btn btn-sm btn-info" id="button_show_hospital_id_'+data.super_admin_id+'" onclick="GetHospitalToDisplayInfo(\''+data.super_admin_id+'\');">Info</button>');
  $("#user_info_dialog").dialog("open");
}

function DisplayUserError(){
  $("#user_info_dialog").html("<h1>Failed loading the information</h1>");
  $("#user_info_dialog").dialog("open");
}
