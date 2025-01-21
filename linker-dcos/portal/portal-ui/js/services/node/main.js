define(['app', '../cluster/main'], function(app) {
	'use strict';
	app.provide.factory('NodeService', ['$http', '$q', '$uibModal', 'ClusterService', function($http, $q, $uibModal, ClusterService) {
		return {
			getNodes: function(cluster_id, skip, limit) {
				var deferred = $q.defer();
				if (_.isUndefined(cluster_id)) cluster_id = "";
				if (_.isUndefined(skip)) skip = "";
				if (_.isUndefined(limit)) limit = "";
				var url = "/cluster/" + cluster_id + "/hosts?count=true&skip=" + skip + "&limit=" + limit;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET"
				}
				$http(request).success(function(data, status, headers, config) {
					data.config=config;
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			getNode:function(cluster_id,host_id){
				var deferred = $q.defer();
				var url = "/cluster/" + cluster_id + "/hosts/"+host_id;
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
			getTags:function(host_id){
				var deferred = $q.defer();
				// var url = "http://52.78.195.159:10030/v1/cmi/server_tag/" + host_id;
				var url = "/v1/cmi/server_tag/" + host_id;
				var request = {
					"url": url,
					"dataType": "json",
					"withCredentials": true,
					"method": "GET"
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			filterTag: function (filter){
				var deferred = $q.defer();
				// http://104.199.196.136:10030/v1/cmi/nodes?server_tag=attacked
				var url = "/v1/cmi/nodes/tag/" + filter + "?t=" + new Date().valueOf();
				var request = {
					"url": url,
					"dataType": "json",
					"withCredentials": true,
					"method": "GET"
				};
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			feedbackTags: function(feedback){
				var deferred = $q.defer();
				var url = "/v1/cmi/nodes/feedback";
				var request = {
					"url": url,
					"dataType": "json",
					"withCredentials": true,
					"method": "POST",
					"data": angular.toJson(feedback)
				};
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			retrain: function (filterItem){
				var deferred = $q.defer();
				var url = "/v1/cmi/nodes/retrain/" + filterItem;
				var request = {
					"url": url,
					"dataType": "json",
					"withCredentials": true,
					"method": "POST"
				};
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			getContainers:function(clientAddr,host_ip, skip, limit){
				var deferred = $q.defer();
				var url = "/containers?skip="+skip+"&limit="+limit+"&host_ip="+host_ip;
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
			createNode:function(node){
				var deferred = $q.defer();
				var url = "/cluster/" + node.clusterId + "/hosts";
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					"data": angular.toJson(node)
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			terminateNode: function(node) {
				var deferred = $q.defer();
				var url = "/cluster/" + node.cluster_id + "/hosts";
				var request = {
					url: url,
					method: 'DELETE',
					data: {
						host_ids:node._id
					},
					headers: {
						"Content-Type": "application/json;charset=utf-8"
					}
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			deleteConfirm : function($scope){
				$uibModal.open({
				    templateUrl: 'templates/node/confirm.html',
				    controller: 'ConfirmController',
				    size: 'sm',
				    backdrop:'static',
				    resolve: {
				        model: function () {
				            return $scope.confirm;
				        }
				    }
			   });
			}
		}
	}]);
});
