function GetHospitalToDisplayInfo(identity){
  var json_val = {"identity":identity};
  $.ajax
    ({
        type: "POST",
        url: '/info/hospital',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: DisplayHospitalInfoInDialog,
        failure:DisplayHospitalInDialogError,
        error:DisplayHospitalInDialogError,
        contentType: 'application/json'
    });
}

function DisplayHospitalInfoInDialog(data){
  $("#hospital_info_hospital_name").text(data.hospital_name);
  $("#hospital_info_super_admin_name").html(data.super_admin_name+'  <button class="btn btn-sm btn-info" id="button_show_hospital_super_admin_id_'+data.super_admin_id+'" onclick="GetUserInfo(\''+data.super_admin_id+'\');">Info</button>');
  if(data.is_created){
    $("#hospital_info_status").html("<span class='text-success'>Approved</span>");
  }else{
    $("#hospital_info_status").html("<span class='text-warning'>Not Approved</span>");
  }
  $("#hospital_info_dialog").dialog("open");
}

function DisplayHospitalInDialogError(){
  $("#hospital_info_dialog").html("<h1>Failed loading the information</h1>");
  $("#hospital_info_dialog").dialog("open");
}
