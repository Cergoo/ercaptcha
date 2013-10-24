


function captcha_get() {
$.ajax({
        url: 'http://localhost:9998/captcha/get',        
        type: "GET",
        cache: 'false',
        processData: 'false',
        dataType: 'jsonp',
        success: function(data) {
        //alert("ljij");                      
          document.getElementById('captchaimg').src = "data:image/png;base64,"+data.img;
          document.getElementById('captchaid').value=data.id;
          document.getElementById('captchaval').value="";            
        }        
    });
}

jQuery(document).ready(function(){
  captcha_get();
})