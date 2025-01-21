define(['login','services/login/main'], function (app) {
    'use strict';
     app.controllerProvider.register('LoginController', ['$scope','$uibModal','$window','$cookies','LoginService','ResponseService',function ($scope,$uibModal,$window,$cookies,LoginService,ResponseService) {
    	  $scope.user = {"username":"","password":""};
    	  $scope.login = function(){
           LoginService.doLogin($scope.user).then(function(data){
               var expireDate = new Date();
               expireDate.setHours(expireDate.getHours() + 6);
               // Setting a cookie
               $cookies.put('username',$scope.user.username,{'expires': expireDate});
               $window.location = "index.html";
             },
              function(error){
				         ResponseService.errorResponse(error,"user.loginFailed");
			       })
    	  };

        setTimeout(function(){
          if($scope.user.username.length>0){
            $("button").prop('disabled', false);
          }
        },800)
     }]);


});
