define(['app'], function(app) {
    'use strict';
    app.provide.factory('ComponentService', ['$http', '$q', function($http, $q) {
        return {
            componentJSON: function() {
                return {
                    JSONTemplate: {
                        "marathon_app": {
                            "id": "",
                            "cmd": "",
                            "args": [],
                            "uris": [],
                            "executor": "",
                            "acceptedResourceRoles": [],
                            "instances": 1,
                            "cpus": "",
                            "gpus": 0,
                            "mem": "",
                            "dependencies": [],
                            "labels": {},
                            "container": {
                                "type": "DOCKER",
                                "docker": {
                                    "image": "",
                                    "network": "BRIDGE",
                                    "parameters":[],
                                    "portMappings": [],
                                    "privileged": false,
                                    "forcePullImage": false
                                },
                                "volumes": []
                            },
                            "env": {},
                            "constraints": [],
                            "healthChecks": [],
                        }
                    },
                    portMapping: {
                        "containerPort": 0,
                        "hostPort": 0,
                        "servicePort": 0,
                        "name": "",
                        "protocol": "tcp"
                    },

                    volume: {
                        "containerPath": "",
                        "hostPath": "",
                        "mode": "RO"
                    },
                    COMMAND: {
                        "protocol": "COMMAND",
                        "command": {
                            "value": ""
                        },
                        "gracePeriodSeconds": 300,
                        "intervalSeconds": 60,
                        "timeoutSeconds": 20,
                        "maxConsecutiveFailures": 3,
                        "ignoreHttp1xx": false
                    },
                    HTTP: {
                        "protocol": "HTTP",
                        "path": "",
                        "portIndex": 0, // or port
                        // "port": 0,
                        "gracePeriodSeconds": 300,
                        "intervalSeconds": 60,
                        "timeoutSeconds": 20,
                        "maxConsecutiveFailures": 3,
                        "portType": "PORT_INDEX",
                        "ignoreHttp1xx": false
                    },
                    TCP: {
                        "protocol": "TCP",
                        "portIndex": 0, // or port
                        // "port": 0,
                        "gracePeriodSeconds": 300,
                        "intervalSeconds": 60,
                        "timeoutSeconds": 20,
                        "maxConsecutiveFailures": 3,
                        "portType": "PORT_INDEX"
                    }
                }


            },
            goNext : function(current) {
                var stepObj = {};
                switch (current) {
                    case "basic":
                        stepObj = { "previous": "basic", "current": "network", "next": "relationship" };
                        break;
                    case "network":
                        stepObj = { "previous": "network", "current": "relationship", "next": "healthCheck" };
                        break;
                    case "relationship":
                        stepObj = { "previous": "relationship", "current": "healthCheck", "next": "advanced" };
                        break;
                    case "healthCheck":
                        stepObj = { "previous": "healthCheck", "current": "advanced", "next": "" };
                        break;
                    case "advanced":
                        stepObj = { "previous": "healthCheck", "current": "advanced", "next": "" };
                        break;
                    default:
                        stepObj = { "previous": "", "current": "basic", "next": "network" };
                }
                return stepObj;
            },
             goPrevious : function(current) {
             	var stepObj = {};
                switch (current) {
                    case "basic":
                        stepObj = { "previous": "", "current": "basic", "next": "network" };
                        break;
                    case "network":
                        stepObj = { "previous": "", "current": "basic", "next": "network" };
                        break;
                    case "relationship":
                        stepObj = { "previous": "basic", "current": "network", "next": "relationship" };
                        break;
                    case "healthCheck":
                        stepObj = { "previous": "network", "current": "relationship", "next": "healthCheck" };
                        break;
                    case "advanced":
                        stepObj = { "previous": "relationship", "current": "healthCheck", "next": "advanced" };
                        break;
                    default:
                        stepObj = { "previous": "", "current": "basic", "next": "network" };
                }
                return stepObj;
            },
            setCurrent : function(current) {
                var stepObj = {};
                switch (current) {
                    case "basic":
                        stepObj = { "previous": "", "current": "basic", "next": "network" };
                        break;
                    case "network":
                        stepObj = { "previous": "basic", "current": "network", "next": "relationship" };
                        break;
                    case "relationship":
                        stepObj = { "previous": "network", "current": "relationship", "next": "healthCheck" };
                        break;
                    case "healthCheck":
                        stepObj = { "previous": "relationship", "current": "healthCheck", "next": "advanced" };
                        break;
                    case "advanced":
                        stepObj = { "previous": "relationship", "current": "advanced", "next": "" };
                        break;
                    default:
                        stepObj = { "previous": "", "current": "basic", "next": "network" };
                }
                return stepObj;
            },

            // getDetail: function(clientAddr, componentname, servicename) {
            //     var deferred = $q.defer();
            //     var url = "/component";
            //     var request = {
            //         "url": url,
            //         "dataType": "json",
            //         "method": "GET",
            //         "params": {
            //             "clientAddr": clientAddr,
            //             "appset_name": servicename,
            //             "name":componentname
            //         }
            //     }

            //     $http(request).success(function(data) {
            //         deferred.resolve(data);
            //     }).error(function(error) {
            //         deferred.reject(error);
            //     });
            //     return deferred.promise;
            // },
            save: function(clientAddr, data, type) {
                var deferred = $q.defer();
                var method = type == "create" ? "POST" : "PUT";
                var url = "/component";
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": method,
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
            scale: function(clientAddr, data) {
                var deferred = $q.defer();
                var url = "/component/scale";
                var scaleto = data.operationType == "increase" ? parseInt(data.currentNum) + parseInt(data.increaseNum) : parseInt(data.currentNum) - parseInt(data.decreaseNum);
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": "PUT",
                    "params": {
                        "clientAddr": clientAddr,
                        "scaleto": scaleto,
                        "name":data.component
                    }

                }

                $http(request).success(function(data) {
                    deferred.resolve(data);
                }).error(function(error) {
                    deferred.reject(error);
                });
                return deferred.promise;
            },
            startOrStop: function(clientAddr, data, type) {
                var deferred = $q.defer();
                var url = "/component/" + type;
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": "PUT",
                    "params": {
                        "clientAddr": clientAddr,
                        "name":data
                    }

                }

                $http(request).success(function(data) {
                    deferred.resolve(data);
                }).error(function(error) {
                    deferred.reject(error);
                });
                return deferred.promise;
            },
            delete: function(clientAddr, data) {
                var deferred = $q.defer();
                var url = "/component";
                var request = {
                    "url": url,
                    "dataType": "json",
                    "method": "DELETE",
                    "params": {
                        "clientAddr": clientAddr,
                        "name":data
                    }

                }

                $http(request).success(function(data) {
                    deferred.resolve(data);
                }).error(function(error) {
                    deferred.reject(error);
                });
                return deferred.promise;
            },
            handleSpecialItem : function(item){
               if(angular.isArray(item)){
                  item = item.length == 0 ? null : item;
               }else if(angular.isObject(item)){
                  item = angular.equals(item,{}) ? null : item;
               }else if(angular.isString(item)){
                  item = item || null;
               }
               return item;
            }



        }

    }]);
});
