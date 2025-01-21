define(['app'], function(app) {
	'use strict';
	app.provide.factory('ClusterInfoService', ['$http', '$q', '$uibModal', function($http, $q, $uibModal) {
		return {
      getTotalCpu: function() {
        var deferred = $q.defer();
        var url = "/v1/cmi/usage/total/cpu";
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
      getTotalMem: function() {
        var deferred = $q.defer();
        var url = "/v1/cmi/usage/total/mem";
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
      getPredictCpu: function() {
        var deferred = $q.defer();
        var url = "/v1/cmi/totaltrend/cpu";
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
      getPredictMem: function(){
        var deferred = $q.defer();
        var url = "/v1/cmi/totaltrend/mem";
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
			getThresholdCpu: function(){
        var deferred = $q.defer();
        var url = "/v1/cmi/threshold/total/cpu";
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
			getThresholdMem: function(){
        var deferred = $q.defer();
        var url = "/v1/cmi/threshold/total/mem";
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
			getDiskusage: function(){
        var deferred = $q.defer();
        var url = "/v1/cmi/diskusage";
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
      getAlertCpu: function() {
        var deferred = $q.defer();
        var url = "/v1/cmi/alarm/total/cpu";
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
      getAlertMem: function() {
        var deferred = $q.defer();
        var url = "/v1/cmi/alarm/total/mem";
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
