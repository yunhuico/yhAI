define(['app'], function(app) {
	'use strict';
	app.provide.factory('PlatformService', ['$http', '$q', function($http, $q) {
		return {
			createProvider: function(provider) {
				switch (provider.type) {
					case "openstack":
						delete provider.awsEc2Info;
						delete provider.googleInfo;
						break;
					case "amazonec2":
						delete provider.openstackInfo;
						delete provider.googleInfo;
						break;
					case "google":
						delete provider.awsEc2Info;
						delete provider.openstackInfo;
						break;
				}

				var deferred = $q.defer();
				var url = "/provider";
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					"data": angular.toJson(provider),
				}

				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			getProvider: function(limit, skip) {
				var deferred = $q.defer();
				var url = "/provider?count=true&skip=" + skip + "&limit=" + limit;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET"
				}

				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			queryProvider: function(id) {
				var deferred = $q.defer();
				var url = "/provider/"+id;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET"
				}

				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			editProvider: function(provider) {
				var deferred = $q.defer();
				switch (provider.type) {
					case "openstack":
						delete provider.awsEc2Info;
						delete provider.googleInfo;
						break;
					case "amazonec2":
						delete provider.openstackInfo;
						delete provider.googleInfo;
						break;
					case "google":
						delete provider.awsEc2Info;
						delete provider.openstackInfo;
						break;
				}
				var url = "/provider/" + provider._id;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "PUT",
					"data":angular.toJson(provider)
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			deleteProvider: function(provider) {
				var deferred = $q.defer();
				var url = "/provider/" + provider._id;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "DELETE"
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			validateName : function(name){
				var deferred = $q.defer();
				var url = "/provider/validate?provider_name=" + name;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET"
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