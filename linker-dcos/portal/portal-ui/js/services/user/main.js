define(['app'], function(app) {
    'use strict';
    app.provide.factory('UserService', ['$http', '$q', function($http, $q) {
        return {
            getUsers: function(skip, limit) {
                var deferred = $q.defer();
                var url = "/user?count=true&skip=" + skip + "&limit=" + limit;
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
            saveUser: function(user, type) {
                var deferred = $q.defer();
                var method = type === "create" ? "POST" : "PUT";
                var url = type === "create" ? "/user" : "/user/" + user._id;
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": method,
                    "data": angular.toJson(user),
                }

                $http(request).success(function(data) {
                    deferred.resolve(data);
                }).error(function(error) {
                    deferred.reject(error);
                });
                return deferred.promise;
            },
            deleteUser: function(user) {
                var deferred = $q.defer();
                var url = "/user/" + user._id;
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
            getUserInfo: function() {
                var deferred = $q.defer();
                var url = "/user/profile";
                var request = {
                    "url": url,
                    "method": "GET"
                }

                $http(request).success(function(data) {
                    deferred.resolve(data);
                }).error(function(error) {
                    deferred.reject(error);
                });
                return deferred.promise;
            },
           
            updatePasssword: function(passwords) {
                var url = "/user/profile/changepassword";
                var deferred = $q.defer();
                var request = {
                    "url": url,
                    "method": "PUT",
                    "data": angular.toJson(passwords)
                }

                $http(request).success(function(data) {
                    deferred.resolve(data);
                }).error(function(error) {
                    deferred.reject(error);
                });
                return deferred.promise;
            },
        }

    }]);
});
