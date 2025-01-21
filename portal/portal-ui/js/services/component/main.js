define(['app'], function (app) {
  'use strict';
   app.provide.factory('ComponentService', ['$http','$q',function ($http,$q) {
      return {
        getComponents : function(clusterId){
          var deferred = $q.defer();
          var url = "/components";
          var request = {
            url: url,
            dataType: "json",
            method: "GET",
            params: {
              clusterId: clusterId
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