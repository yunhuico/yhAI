define(['app'], function (app) {
  'use strict';
  app.provide.factory('AlertService', ['$http','$q',function ($http,$q) {
    return {
      getAlerts : function(clientAddr, params){
        var deferred = $q.defer();
        var url = "/alerts?count=true&skip=" + params.skip + "&limit=" + params.limit;
        var request = {
          "url": url,
          "dataType": "json",
          "method": "GET",
          "params": {
            "clientAddr": clientAddr,
            "alert_name": params.alert_name,
            "action": params.action
          }
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