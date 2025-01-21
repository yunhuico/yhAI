define(['app', 'directives/pagination', 'services/network/main', 'services/node/main'], function(app) {
	'use strict';
	app.controllerProvider.register('NetworkController', ['$scope', '$localStorage', '$uibModal', '$location', '$state', 'NetworkService', 'CommonService', 'ResponseService', function($scope, $localStorage, $uibModal, $location, $state, NetworkService, CommonService, ResponseService) {
		$scope.$storage = $localStorage;
		$scope.endPointAvailable = CommonService.endPointAvailable();
		$scope.addNetwork = function() {
			if ($scope.endPointAvailable) {
				$uibModal.open({
						templateUrl: 'templates/network/add.html',
						controller: 'AddNetworkController',
						backdrop: 'static',
						resolve: {
							model: function() {
								return {
									users: $scope.users,
									types: $scope.types
								};
							}
						}
					})
					.result
					.then(function(response) {
						if (response.operation === 'execute') {
							console.log(response.data);

							NetworkService.createNetwork(CommonService.getEndPoint(), response.data).then(function(data) {
									if(data.data.network.internal=="true"){
										if($scope.checkdata.internal==true){
											$scope.getNetwork(true);
										}else{
											$scope.changeInternal(true);
										}
									}else{
										if($scope.checkdata.internal==false){
											$scope.getNetwork(false);
										}else{
											$scope.changeInternal(false);
										}
									}
								},
								function(errorMessage) {
									ResponseService.errorResponse(errorMessage);
								});
						}
					});
			}
		};
		$scope.terminateAll = function() {
			$scope.$translate(['common.deleteConfirm', 'network.terminateAllConfirm', 'common.delete']).then(function(translations) {
				$scope.confirm = {
					"title": translations['common.deleteConfirm'],
					"message": translations['network.terminateAllConfirm'],
					"button": {
						"text": translations['common.delete'],
						"action": function() {
							NetworkService.terminateAll(CommonService.getEndPoint(), $scope.$storage.cluster._id).then(function(data) {
								$scope.getNetwork(true);
							}, function(errorMessage) {
								ResponseService.errorResponse(errorMessage);
							})
						}
					}
				};
				CommonService.deleteConfirm($scope);
			});
		};
		$scope.terminateNetwork = function(item, internal) {
			$scope.$translate(['common.deleteConfirm', 'network.terminateConfirm', 'common.delete']).then(function(translations) {
				$scope.confirm = {
					"title": translations['common.deleteConfirm'],
					"message": translations['network.terminateConfirm'],
					"button": {
						"text": translations['common.delete'],
						"action": function() {
							NetworkService.terminateNetwork(CommonService.getEndPoint(), item._id).then(function(data) {
								$scope.getNetwork(internal);
							}, function(errorMessage) {
								ResponseService.errorResponse(errorMessage);
							})
						}
					}
				};
				CommonService.deleteConfirm($scope);
			});
		}
		$scope.getNetwork = function() {

			var skip = ($scope.checkdata.currentPage - 1) * $scope.recordPerPage;
			var limit = $scope.recordPerPage;
			if ($scope.endPointAvailable) {
				NetworkService.getNetwork(CommonService.getEndPoint(), skip, limit, $scope.$storage.cluster._id).then(function(data) {
						$scope.network = data.data;
						$scope.totalrecords = data.count;
						$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
					},
					function(errorMessage) {
						ResponseService.errorResponse(errorMessage);
					});
			} else {
				$scope.network = [];
				$scope.totalrecords = 0;
				$scope.totalPage = 0;
			}
		};
		$scope.$watch('$storage.cluster', function(newValue, oldValue) {
			if (!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id) {
				$scope.checkdata = {
					currentPage: 1,
					internal: true
				}
				$scope.totalPage = 0;
				$scope.totalrecords = 0;
				$scope.endPointAvailable = CommonService.endPointAvailable();
				$scope.getNetwork($scope.checkdata.internal);
			}

		}, true);
		$scope.$watch('checkdata', function(nv, ov) {
			$scope.getNetwork($scope.checkdata.internal);
		}, true);
		var init = function(internal) {
			$scope.totalPage = '';
			$scope.recordPerPage = CommonService.recordNumPerPage();
			$scope.totalrecords = '';
			$scope.checkdata = {
				currentPage: 1,
				internal: internal
			}
			$scope.activeTab=[internal,!internal]
		};
		init(true);
		$scope.changeInternal = function(internal) {
			init(internal);
		}
	}]);
	app.controllerProvider.register('AddNetworkController', ['$scope', '$cookies', '$uibModalInstance', 'model', 'NetworkService', 'NodeService', '$localStorage', 'CommonService',
		function($scope, $cookies, $uibModalInstance, model, NetworkService, NodeService, $localStorage, CommonService) {
			$scope.validate = true;
			$scope.$storage = $localStorage;
			$scope.users = model.users;
			$scope.types = model.types;
			$scope.driverTypes = [
				{
					"name": "overlay",
					"value": "overlay",
					"checked": "selected",
					"bridge_name": "sgw",
					"bridge_value": "sgw",
				},
				{
					"name": "pipework",
					"value": "ovs",
					"checked": "",
					"bridge_name": "pgw",
					"bridge_value": "pgw",
				}
			];
			$scope.network = {
				"cluster_id": $scope.$storage.cluster._id,
				"cluster_name": $scope.$storage.cluster.name,
				"user_name": $scope.$storage.cluster.owner,
				"clust_host_name": '',
				"network": {
					"name": "",
					"internal": "false",
					"driver": "overlay",
					"subnet": [],
					"gateway": [],
					"iprange": [],
				}
			};
			$scope.mask=false;
			$scope.$watch('network.network', function(nv, ov) {
				if (nv.name == ov.name && nv.internal == ov.internal && nv.driver == ov.driver) {
					var subnet,subnetlength,gateway,iprange,iprangelength;
					if (!_.isEmpty(nv.subnet[0])) {
						subnetlength = nv.subnet[0].slice(nv.subnet[0].indexOf('/') + 1);
						subnet = nv.subnet[0].slice(0, nv.subnet[0].indexOf('/')).split('.');
						subnet = _.map(subnet, function(item) {
							item = parseInt(item).toString(2);
							var len = item.length;
							while (len < 8) {
								item = "0" + item;
								len++;
							}
							return item;
						});
						subnet = subnet.join('');
						$scope.subnetcheck=(subnet.slice(subnetlength)!=0)
					}else{
						$scope.subnetcheck=false;
					}
					if (!_.isEmpty(nv.gateway[0])) {
						gateway = nv.gateway[0].split('.');
						gateway = _.map(gateway, function(item) {
							item = parseInt(item).toString(2);
							var len = item.length;
							while (len < 8) {
								item = "0" + item;
								len++;
							}
							return item;
						});
						gateway = gateway.join('');
					}
					if (!_.isEmpty(nv.iprange[0])) {
						iprangelength = nv.iprange[0].slice(nv.iprange[0].indexOf('/') + 1);
						iprange  = nv.iprange[0].slice(0, nv.iprange[0].indexOf('/')).split('.');
						iprange = _.map(iprange, function(item) {
							item = parseInt(item).toString(2);
							var len = item.length;
							while (len < 8) {
								item = "0" + item;
								len++;
							}
							return item;
						});
						iprange = iprange.join('');
					}
					if(subnetlength&&iprangelength&&parseInt(subnetlength)>parseInt(iprangelength)){
						$scope.mask=true;
						return;
					}else{
						$scope.mask=false;
					}
					if(subnet&&gateway){
						if(subnet.slice(0,subnetlength)==gateway.slice(0,subnetlength)){
							$scope.gatewaysame=false;
						}else{
							$scope.gatewaysame=true;
						}
					}
					if(subnet&&iprange){
						if(subnet.slice(0,subnetlength)==iprange.slice(0,subnetlength)){
							$scope.iprangesame=false;
						}else{
							$scope.iprangesame=true;
						}
					}
				}
			}, true);
			$scope.validateName = function() {
				var networkname = $scope.network.network.name;
				var username = $cookies.get('username');
				if (!angular.isUndefined(networkname) && !_.isEmpty(networkname)) {
					NetworkService.checkname(CommonService.getEndPoint(), username, networkname).then(function(data) {
							$scope.validate = true;
						},
						function(errorMessage) {
							$scope.validate = false;
						});
				} else {
					$scope.validate = true;
				}

			};
			NodeService.getNodes($scope.network.cluster_id).then(function(data) {
				$scope.network.sharedClusters = _.filter(data.data, function(item) {
					if (item.isSharedNode || item.isSlaveNode)
						return item;
				});
			}, function(errorMessage) {
				ResponseService.errorResponse(errorMessage);
			});
			$scope.close = function(res) {
				// when select overlay, random give a shared cluster node.
				// console.log("$scope.network.sharedClusters ---");
				// console.log($scope.network.sharedClusters);
				if ($scope.network.network.driver === "overlay") {
					$scope.network.clust_host_name = $scope.network.sharedClusters[0].hostName;
				}
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.network
				});
			};
		}
	]);
});
