function GetUserInfo(){
  var curr_url_string = window.location.href;
  var curr_url = new URL(curr_url_string);
  var identity = curr_url.searchParams.get("id");
  var json_val = {"identity":identity};
  $.ajax
    ({
        type: "POST",
        url: '/info/user',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: DisplayUserInfo,
        failure:DisplayError,
        error:DisplayError,
        contentType: 'application/json'
    });
}

function DisplayUserInfo(data){
  $("#name").text(data.name);
  $("#id").text(data.id);
  switch (data.role) {
    case "super_admin":
      $("#role").text("Head of Hospital");
      break;
    case "admin":
      $("#role").text("Administrator");
      break;
    case "manager":
      $("#role").text("Manager");
      break;
    case "user":
      $("#role").text("User");
      break;
    default:
      $("#role").text("Unknown");
  }
  if(data.is_created){
    $("#status").text("Approved");
    $("#darc").html("<a target='_blank' href='/gui/darc?base_id="+data.darc_base_id+"'>See details</a>");
  }else{
    $("#status").text("Not Approved");
    $("#darc").text("Not Created Yet");
  }
  $("#hospital").html("<a target='_blank' href='/gui/hospital?id="+data.super_admin_id+"'>"+data.hospital_name+"</a>");
}

function DisplayError(){
  $("body").html("<h1>Failed loading the information</h1>")
}


$(document).ready(
  function(){
    GetUserInfo();
  }
);
