define(['app'], function(app) {
	'use strict';
	app.provide.factory('ClusterService', ['$http', '$q', '$uibModal', '$cookies', function($http, $q, $uibModal, $cookies) {
		return {
			getClusters: function(skip, limit, status, user_id) {
				var deferred = $q.defer();
				if (_.isUndefined(user_id)) user_id = "";
				if (_.isUndefined(skip)) skip = "";
				if (_.isUndefined(limit)) limit = "";
				var url = "/cluster?count=true&skip=" + skip + "&limit=" + limit + "&user_id=" + user_id + "&status=" + status;
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
			getCluster:function(id){
				var deferred=$q.defer();
				var url = "/cluster/"+id;
				var request={
					"url":url,
					"dataType":"json",
					"method":"GET"
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			createCluster: function(data) {
				var deferred = $q.defer();
				var url = "/cluster";
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					"data": angular.toJson(data)
				}
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			terminateCluster: function(clusterid) {
				var deferred = $q.defer();
				var url = "/cluster/"+clusterid;
				var request = {
					"url": url,
					"method": "DELETE",
					"headers": {
						"Content-Type": "application/json;charset=utf-8"
					}
				};
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			validateClusterForUser: function(clustername) {
				var deferred = $q.defer();
				var url = '/clusterValidate';
				var request = {
					"url": url,
					"dataType": "json",
					"method": "GET",
					"params": {
						"clustername": clustername,
					}
				};
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
			},
			getClusterSetting: function( clusterid, cmi ) {
				var deferred = $q.defer();
				var url = "/cluster/" + clusterid + "/setting";
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					data: {
						cmi: cmi
					}
				};

				console.log( request );
				$http(request).success(function(data) {
					deferred.resolve(data);
				}).error(function(error) {
					deferred.reject(error);
				});
				return deferred.promise;
			},
			getNetworkOvs: function (clientAddr, data) {
				var deferred = $q.defer();
				var clusterid = data.clusterid;
				var hostNames = data.hostNames;
				var url = "/network/ovs?clusterid=" + clusterid + "&clientAddr=" + clientAddr;
				var request = {
					"url": url,
					"dataType": "json",
					"method": "POST",
					data: {
						host_names: hostNames
					}
				};

				console.log( request );
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
