define(['app'], function (app) {
    'use strict';
     app.provide.factory('KeyPairService', ['$http','$q',function ($http,$q) {
	    	return {
                getKeyPairs : function(skip, limit){
                	var skip = skip || 0;
                	var limit = limit || 1000;
					var deferred = $q.defer();
					var url = "/keypair?t=" + new Date().valueOf() + "&count=true&skip="+skip+"&limit="+limit;
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
			    createKeyPair : function(keypair){
			    	var deferred = $q.defer();
					var url = "/keypair/create";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"data" : angular.toJson(keypair),
					}

					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    downloadKeyPair : function(userid){
					var deferred = $q.defer();
					var url = "/keypair/download/"+userid;
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
			    uploadKeyPair : function(keypair){
			    	var deferred = $q.defer();
					var url = "/keypair/upload";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"data" : angular.toJson(keypair),
					}

					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    deleteKeyPair : function(keypair){
			    	var deferred = $q.defer();
					var url = "/keypair/"+keypair._id;
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
		        addPublickeyToCluster : function(dockerIds, clusterId){
		        	var deferred = $q.defer();
		    		var url = "/v1/cluster/" + clusterId + "/pubkey";
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
		        deletePublickeyToCluster : function(dockerIds, clusterId){
		        	var deferred = $q.defer();
		    		var url = "/v1/cluster/" + clusterId + "/pubkey";
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
