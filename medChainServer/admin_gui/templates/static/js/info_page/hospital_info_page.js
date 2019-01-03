function GetHospitalInfo(){
  var curr_url_string = window.location.href;
  var curr_url = new URL(curr_url_string);
  var identity = curr_url.searchParams.get("id");
  var json_val = {"identity":identity};
  $.ajax
    ({
        type: "POST",
        url: '/info/hospital',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: DisplayHospitalInfo,
        failure:DisplayError,
        error:DisplayError,
        contentType: 'application/json'
    });
}

function DisplayHospitalInfo(data){
  $("#hospital_name").text(data.hospital_name);
  $("#super_admin_name").html("<a target='_blank' href='/gui/user?id="+data.super_admin_id+"'>"+data.super_admin_name+"</a>");
  if(data.is_created){
    $("#status").text("Approved");
  }else{
    $("#status").text("Not Approved");
  }
}

function DisplayError(){
  $("body").html("<h1>Failed loading the information</h1>")
}


$(document).ready(
  function(){
    GetHospitalInfo();
  }
);
