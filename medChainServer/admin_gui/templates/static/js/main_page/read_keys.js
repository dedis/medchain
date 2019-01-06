function handleKeyFileSelect(input_name, callback_success, callback_error){
  if (!window.File || !window.FileReader || !window.FileList || !window.Blob) {
    ShowLoginError({"responseText":'The File APIs are not fully supported in this browser.'});
    return;
  }

  var input = document.getElementById(input_name);
  if (!input) {
    callback_error({"responseText":'You have to select a public key.'});
  }
  else if (!input.files) {
    callback_error({"responseText":"This browser doesn't seem to support the `files` property of file inputs."});
  }
  else if (!input.files[0]) {
    callback_error({"responseText":"Please select a public key before logging in"});
  }
  else {
    file = input.files[0];
    var fr = new FileReader();
    fr.onload = function(){
      var key = fr.result;
      document.getElementById(input_name).value = "";
      callback_success(key);
    };
    fr.readAsText(file);
  }
}
