define(['app'], function(app) {
	'use strict';
	app.provide.factory('NetworkService', ['$http', '$q', function($http, $q) {
		return {
			getNetwork: function(clientAddr, skip, limit,clusterId) {
				var deferred = $q.defer();
				var url = "/network?skip=" + skip + "&limit=" + limit+"&cluster_id="+clusterId;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET",
					"params": {
						"clientAddr": clientAddr
					}
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			createNetwork: function(clientAddr,data) {
				var deferred = $q.defer();
				var url = "/network";
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					"params": {
						"clientAddr": clientAddr
					},
					"data": angular.toJson(data)
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			terminateNetwork: function(clientAddr,networkid) {
				var deferred = $q.defer();
				var url = "/network/" + networkid;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "DELETE",
					"params": {
						"clientAddr": clientAddr
					}
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			terminateAll:function(clientAddr,clusterid){
				var deferred = $q.defer();
				var url = "/network?cluster_id=" + clusterid;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "DELETE",
					"params": {
						"clientAddr": clientAddr
					}
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			checkname:function(clientAddr,username,networkname){
				var deferred = $q.defer();
				var url = "/network/validate?username=" + username+'&networkname='+networkname;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET",
					"params": {
						"clientAddr": clientAddr
					}
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			}
		}
	}]);
});