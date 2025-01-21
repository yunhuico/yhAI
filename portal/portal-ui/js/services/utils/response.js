define(['services/module'], function (app) {
    'use strict';
     app.factory('ResponseService', ['$window', '$uibModal','$cookies',function ($window,$uibModal,$cookies) {
            return {
			  errorResponse : function(error,title){
				if(angular.isUndefined(error) || error == null){
					error = {"code":"Web.ServiceUnavailable"};
				}else if(angular.isString(error)){
					error = {"code":"500"};
				}else if(error.code==''){
                    error = {"code":"Web.UnreachableServer"};
				}

				if(error.code == 400 && error.data['type'] == "RepositoryAlreadyPresent") {
					error = {"code":"repositoryAlreadyPresent"};
				}
				title = title || "common.failed";				
				$uibModal.open({
				        templateUrl: 'templates/common/failed.html',
				        controller: 'ActionFailedController',
				        size: 'sm',
				        backdrop:'static',
				        resolve: {
				          model: function () {
				            return {
				              message:{"title":title,"content":error}
				            }
				          }
				        }
			      })
			      .result
			      .then(function (result) {
			           if(error.code === "E10012" || error.code === "Web.NotSignIn" || error.code === "E10050" || error.code === "E10052" || error.code ==="E10016"){	  
		                   $cookies.remove('username');
		                   $window.location = "login.html";
			           }
			      });
			  },

			  checkSession : function(){
				if(_.isUndefined($cookies.get('username')) || _.isEmpty($cookies.get('username'))){
						return false;
					}else{
					    return true;	   
					}
			    }
	    	 }
	    	
     }]);
});
