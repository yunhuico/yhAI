define(['app'], function (app) {
    'use strict';
     app.provide.factory('LogService', ['$http','$q',function ($http,$q) {
	    	return {
                getLogs : function(clusterid,skip, limit){
					var deferred = $q.defer();
					var url = "/logs?count=true&skip="+skip+"&limit="+limit+"&clusterid="+clusterid;
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
			    }


	    	}
	    	
     }]);
});