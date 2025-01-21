define(['app'], function (app) {
    'use strict';
     app.provide.factory('MonitorService', ['$http','$q',function ($http,$q) {
	    	return {
	    		getNodes : function(nginxip){
					var deferred = $q.defer();
					var url = "/monitor/slavenodes";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": nginxip
					    }
					}

					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
				},
	    		getContainers : function(nginxip){
					var deferred = $q.defer();
					var url = "/monitor/containers";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": nginxip
					    }
					}

					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
				},
				getServices : function(nginxip){
					var deferred = $q.defer();
					var url = "/appsets?count=true&skip_group=1&skip=0&limit=0";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": nginxip
					    }
					}

					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
				},
				getServiceContainers : function(nginxip,groupid){
					var deferred = $q.defer();
					var url = "/monitor/service/containers";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": nginxip,
						  "groupid" : groupid
					    }
					}

					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
				},
               	nodeMonitor : function(ip,slaveid){
					var deferred = $q.defer();
					var url = "/nodemonitoring?ip="+ip+ "&slaveid=" + slaveid;
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
				getPredictCpu : function(hostName){
					var deferred = $q.defer();
					var url = "/v1/cmi/trend/" + hostName + "/cpu";
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
				getPredictMem : function(hostName){
					var deferred = $q.defer();
					var url = "/v1/cmi/trend/" + hostName + "/mem";
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
        getDataCPU : function(hostName){
					var deferred = $q.defer();
					var url = "/v1/cmi/usage/" + hostName + "/cpu";
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
        getDataMem : function(hostName){
					var deferred = $q.defer();
					var url = "/v1/cmi/usage/" + hostName + "/mem";
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
        getThresholdCpu : function(hostName){
					var deferred = $q.defer();
					var url = "/v1/cmi/threshold/" + hostName + "/cpu";
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
        getThresholdMem : function(hostName){
					var deferred = $q.defer();
					var url = "/v1/cmi/threshold/" + hostName + "/mem";
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
				nodeSpec : function(ip,slaveid){
					var deferred = $q.defer();
					var url = "/nodespec?ip="+ ip + "&slaveid=" + slaveid;
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
				containerMonitor : function(ip,slaveid,dockername){
					var deferred = $q.defer();
					var url = "/containermonitoring?ip=" + ip + "&slaveid=" + slaveid + "&dockername=" + dockername;
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
				containerSpec : function(ip,slaveid,dockername){
					var deferred = $q.defer();
					var url = "/containerspec?ip=" + ip + "&slaveid=" + slaveid + "&dockername=" + dockername;
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
				machineInfo : function(ip,slaveid){
					var deferred = $q.defer();
					var url = "/machineinfo?ip=" + ip + "&slaveid=" + slaveid;
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
