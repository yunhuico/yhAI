define(['app'], function (app) {
    'use strict';
     app.provide.factory('ContainerService', ['$http','$q',function ($http,$q) {
	    	return {              
			    get : function(clientAddr,taskid,slaveid){
			    	var deferred = $q.defer();
					var url = "/container/"+taskid;
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": clientAddr,
						  "slaveid":slaveid				  
					    }
					}
						
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    getName : function(clientAddr,slaveid,taskid){
			    	var deferred = $q.defer();
					var url = "/container/cadvisor/name";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": clientAddr,
						  "slaveid":slaveid,
						  "taskid":taskid
					    }
					}
						
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    getContainerLogs : function(clientAddr, slaveid, params){
			    	var deferred = $q.defer();
						var request = {
							"url": "container/read",
							"dataType": "json",
							"method": "GET",
							"params": {
					  		"clientAddr": clientAddr,
					  		"slaveid":slaveid,
					  		"type": params.type,
                "offset": params.offset,
                "length": params.length,
                "volumePath": params.volumePath
				    	}
						}
							
						$http(request).success(function(data){
							deferred.resolve(data);
						}).error(function(error){
							deferred.reject(error);
						});
						return deferred.promise;
			    },
          browseFolder : function(clientAddr, slaveid, path) {
            var deferred = $q.defer();
            var request = {
              "url": "container/browse",
              "dataType": "json",
              "method": "GET",
              "params": {
                "clientAddr": clientAddr,
                "slaveid": slaveid,
                "path": path
              }
            }

            $http(request).success(function(data){
              deferred.resolve(data);
            }).error(function(error){
              deferred.reject(error);
            });
            return deferred.promise;
          },
			    operate : function(clientAddr,taskid,type){
			    	var deferred = $q.defer();
					var url = "/container/"+taskid+"/"+type;
					var request = {
						"url": url,
						"dataType": "json",
						"method": "PUT",
						"params": {
						  "clientAddr": clientAddr					 
					    }
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