define(['app'], function (app) {
    'use strict';
     app.provide.factory('DockerregistriesService', ['$http','$q',function ($http,$q) {
	    	return {
                getDockerregistries : function(skip, limit){
					var deferred = $q.defer();
					var url = "/dockerregistries?t=" + new Date().valueOf() + "&count=true&skip="+skip+"&limit="+limit;		
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
			    createDockerregistry : function(registry){
			    	var deferred = $q.defer();
					var url = "/dockerregistries";	
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"data" : angular.toJson(registry),
					}
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			   },
			   deleteDockerregistry : function(id){
			    	var deferred = $q.defer();
					var url = "/dockerregistries/"+id;		
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
			    },
			    validateName:function(name,type){
			    	var deferred = $q.defer();
					var url = "/registryValidate?name="+name+"&type="+type;		
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
		        addDockerregistryToCluster : function(dockerIds, clusterId){
		        	var deferred = $q.defer();
		    		var url = "/v1/cluster/" + clusterId + "/registry";
		    		var request = {
		    			"url": url,
		    			"dataType": "json",
		    			"method": "POST",
		    			"data" : angular.toJson(dockerIds)
		    		}
		    			
		    		$http(request).success(function(data){
		    			deferred.resolve(data);
		    		}).error(function(error){
		    			deferred.reject(error);
		    		});
		    		return deferred.promise;
		        },
		        deleteDockerregistryToCluster : function(dockerIds, clusterId){
		        	var deferred = $q.defer();
		    		var url = "/v1/cluster/" + clusterId + "/registry";
		    		var request = {
		    			"url": url,
		    			headers: {
		    				"Content-Type": "application/json;charset=utf-8"
		    			},
		    			"method": "DELETE",
		    			"data": dockerIds
		    		};
		    			
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