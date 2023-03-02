

/*
 * validate an e-mail address by leveraging the HTML5
 * input element with type "email"
 */

function validate() {
    var validRegex = /^\w+([\.-]?\w+)*@\w+([\.-]?\w+)*(\.\w{2,3})+$/;
    //var email=document.getElementById('email').value;
      
    if ($('#email').val().match(validRegex)) {
      $('#email').css("background-color","lightgreen")
    } else {
        $('#email').css("background-color","rgb(255, 204, 204)")
    }
      
    return false;
 }

function validateCard(){
   var visaReg=/^(?:4[0-9]{12}(?:[0-9]{3})?)$/;
   console.log($("#ccnum").val())
   console.log($("#cardnum").val().match(visaReg))
   event.preventDefault();
   if($("#cardnum").val().match(visaReg)){
     alert("valid visa")
      console.log("valid")
     return true;
   }else{
     return false;
   }
}