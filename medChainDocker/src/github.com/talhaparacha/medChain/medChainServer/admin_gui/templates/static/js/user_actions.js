function AllowAddUser(){
  $("#add_user_div").show();
}

function AddUser(){
  var public_key = $("#new_user_public_key_input").val();
  var name = $("#new_user_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
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



function AllowAddManager(){
  $("#add_manager_div").show();
}

function AddManager(){
  var public_key = $("#new_manager_public_key_input").val();
  var name = $("#new_manager_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
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


function AllowAddAdmin(){
  $("#add_admin_div").show();
}

function AddAdmin(){
  var public_key = $("#new_admin_public_key_input").val();
  var name = $("#new_admin_name_input").val();
  var json_val = {"initiator":info.signer.id, "name":name, "new_public_key":public_key, "super_admin_id": info.hospital.id, preferred_signers:[info.signer.id]}
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



function AllowAddHospital(){
  $("#add_hospital_div").show();
}

function AddHospital(){
  var public_key = $("#new_hospital_public_key_input").val();
  var hospital_name = $("#new_hospital_name_input").val();
  var super_admin_name = $("#new_hospital_head_name_input").val();
  var json_val = {"initiator":info.signer.id, "hospital_name":hospital_name, "super_admin_name":super_admin_name, "new_public_key":public_key}
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

function AllowAddProject(){
  $("#add_project_div").show();
}

function AddProject(){
  var project_name = $("#new_project_name_input").val();
  var managers = GetNewProjectManagers();
  var queries = GetNewProjectQueryMapping();
  var json_val = {"initiator":info.signer.id, "name":project_name, "managers":managers, "queries":queries}
  $.ajax
    ({
        type: "POST",
        url: '/add/project',
        dataType: 'json',
        data: JSON.stringify(json_val),
        success: AddProjectCallback,
        failure:
        function(error){
          $("#add_project_error").text("Error: "+ error["responseText"]);
        },
        contentType: 'application/json'
    });
}

function GetNewProjectManagers(){
  return ["ed25519:d26dc4c38731fef9d9270b5b1a0149f683ecdd636903b30d3bb6efe38b8002d0"]
}

function GetNewProjectQueryMapping(){
  return {"AggregatedQuery":["ed25519:07bb05922b600fba97d687f314ea5cb0c9dce3268979de470ef65140430066a8"]}
}

function AddProjectCallback(data){
  GetProjectList();
}
