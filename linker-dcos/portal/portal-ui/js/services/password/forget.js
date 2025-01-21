define(['login'], function(app) {
	'user strict';
	app.provide.factory('ForgetPasswordService', ['$http', '$q', function($http, $q) {
		return {
			forgetPassword: function(username, host, port) {
				var deferred = $q.defer();
				var hostPort = host + ':' + port;
				var url = "/forget";		
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					"data":angular.toJson({"username": username, 'ip': hostPort})
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
