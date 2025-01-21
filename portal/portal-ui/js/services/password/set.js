define(['login'], function(app) {
	'user strict';
	app.provide.factory('SetPasswordService', ['$http', '$q', function($http, $q) {
		return {
			resetPassword: function(passwords, username, activecode) {
				var deferred = $q.defer();
				var url = "/forget/change";		
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					"data":angular.toJson({
						"newpassword": passwords.newpassword,
						"username": username,
						"activecode": activecode
					})
				}
					
				$http(request).success(function(data){
					deferred.resolve(data);
				}).error(function(error){
					deferred.reject(error);
				});
				return deferred.promise;
			}
		}
	}])
})
