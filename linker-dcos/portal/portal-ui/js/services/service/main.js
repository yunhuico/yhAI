define(['app'], function(app) {
    'use strict';
    app.provide.factory('ServiceService', ['$http', '$q', function($http, $q) {
        return {
            get: function(clientAddr, skip, limit) {
                var deferred = $q.defer();
                var url = "/appsets?count=true&skip=" + skip + "&limit=" + limit;
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
            getDetail: function(clientAddr, servicename) {
                var deferred = $q.defer();
                if (servicename) {
                    var url = "/appsets/" + servicename + "?skip_group=true";
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
                }

                return deferred.promise;
            },
            save: function(clientAddr, data) {
                try {
                    data.group = data.group == "" ? {} : angular.fromJson(data.group);
                } catch (e) {

                }
                var deferred = $q.defer();
                var url = "/appsets";
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
            update: function(clientAddr, data) {
                try {
                    data.group = data.group == "" ? {} : angular.fromJson(data.group);
                } catch (e) {

                }
                var deferred = $q.defer();
                var url = "/appsets/" + data.name;
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": "PUT",
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
            delete: function(clientAddr, service) {
                var deferred = $q.defer();
                var url = "/appsets/" + service.name;
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
            stop: function(clientAddr, service) {
                var deferred = $q.defer();
                var url = "/appsets/" + service.name + "/stop";
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": "PUT",
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
            start: function(clientAddr, service) {
                var deferred = $q.defer();
                var url = "/appsets/" + service.name + "/start";
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": "PUT",
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
            getapps: function(clientAddr, name) {
                var deferred = $q.defer();
                var url = "/appsets/" + name + "/apps";
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
            webconsole:function(cid,clientAddr){
            	var deferred = $q.defer();
            	var url = "/webconsole?cid="+cid
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
