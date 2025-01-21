define(['app'], function (app) {
    'use strict';
     app.provide.factory('SMTPService', ['$http','$q',function ($http,$q) {
	    	return {
                getSMTPServers : function(skip, limit){
					var deferred = $q.defer();
					var url = "/smtp?count=true&skip="+skip+"&limit="+limit;		
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET"
					}
						
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
					
			    },
			    addSMTPServer : function(smtp){
			    	var deferred = $q.defer();
					var url = "/smtp";	
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"data" : angular.toJson(smtp),
					}
						
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    editSMTPServer : function(smtp){
			    	var deferred = $q.defer();
					var url = "/smtp/"+smtp._id;	
					var request = {
						"url": url,
						"dataType": "json",
						"method": "PUT",
						"data" : angular.toJson(smtp),
					}
						
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    deleteSMTPServer : function(smtp){
			    	var deferred = $q.defer();
					var url = "/smtp/"+smtp._id;		
					var request = {
						"url": url,
						"dataType": "json",
						"method": "DELETE"	
					}
						
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    }
	    	}
	    	
     }]);
});