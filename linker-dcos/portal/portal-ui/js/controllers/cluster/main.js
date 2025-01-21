define(['app', 'services/cluster/main', 'services/user/main', 'services/configuration/platform', 'directives/main', 'directives/pagination', 'directives/ipduplicate','services/configuration/key','services/configuration/dockerregistries'], function(app) {
	'use strict';
	app.controllerProvider.register('ClusterController', ['$scope', '$localStorage', '$location', '$stateParams', '$uibModal', 'ClusterService', 'UserService','DockerregistriesService', 'ResponseService', 'CommonService', 'PlatformService', 'KeyPairService', '$q', function($scope, $localStorage, $location, $stateParams, $uibModal, ClusterService, UserService, DockerregistriesService,ResponseService, CommonService, PlatformService, KeyPairService, $q) {
		var getClusters = function() {
			ClusterService.getClusters('', '', 'unterminated', '').then(function(data) {
					$scope.clusters = data.data;
					$scope.deleteClusters = _.filter($scope.clusters, function(cluster) {
						return cluster.status != 'TERMINATING' && cluster.status != 'INSTALLING' && cluster.status != 'MODIFYING';
					});
					if (!$scope.$storage.cluster) {
						$scope.$storage.cluster = $scope.clusters[0];
					} else {
						if (_.find($scope.clusters, function(item) {
								return item._id == $scope.$storage.cluster._id
							})) {
							$scope.$storage.cluster = _.find($scope.clusters, function(item) {
								return item._id == $scope.$storage.cluster._id
							});
						} else {
							$scope.$storage.cluster = $scope.clusters[0];
						}

					};
					$scope.selectCluster = JSON.stringify($scope.$storage.cluster);
				},
				function(errorMessage) {
					ResponseService.errorResponse(errorMessage);
				});
		}
		$scope.$storage = $localStorage;
		$scope.createCluster = function() {
			var promises = [KeyPairService.getKeyPairs('', ''), PlatformService.getProvider('', ''),DockerregistriesService.getDockerregistries('','')];
			$q.all(promises).then(function(data) {
				$uibModal.open({
						templateUrl: 'templates/cluster/createCluster.html',
						controller: 'createClusterController',
						backdrop: 'static',
						resolve: {
							model: function() {
								return {
									keypairs: data[0].data,
									providers: data[1].data,
									registries: data[2].data
								}
							}
						}
					})
					.result
					.then(function(response) {
						if (response.operation === 'execute') {
							if (!_.isUndefined(response.data.createNode.privateKey)) {
								response.data.createNode.privateKey = Base64.encode(response.data.createNode.privateKey);
							}

							response.data['owner'] = $scope.user.email;
							response.data['user_id'] = $scope.user._id;
							if ( ! response.data['pubkeyId'].length > 0) {
								response.data['pubkeyId'] = [];
							} else {
								response.data['pubkeyId'] = [response.data['pubkeyId']];
							}

							response.data['masterCount'] = 0;
							response.data['sharedCount'] = (response.data['sharedCount'] && response.data['sharedCount'] > 0) ? response.data['sharedCount'] : 0;
							response.data['pureslaveCount'] = (response.data['pureslaveCount'] && response.data['pureslaveCount'] > 0) ? response.data['pureslaveCount'] : 0;
							response.data['sharedNodes'] = [];
							response.data['pureslaveNodes'] = [];
							response.data['masterNodes'] = [];

							var otherNodes = response.data.createNode.nodes;

							for (var i in otherNodes) {
								var node = otherNodes[i];
								node['privateKey'] = response.data.createNode.privateKey;
								node['privateNicName'] = response.data.createNode.privateNicName;

								if (node.type === 'master') {
									response.data['masterNodes'].push(node);
									response.data['masterCount'] += 1;
									continue;
								}

								if ( ! node.type) {
									response.data['sharedNodes'].push(node);
									response.data['sharedCount'] += 1;
									continue;
								}

								if (node.type === 'share') {
									response.data['sharedNodes'].push(node);
									response.data['sharedCount'] += 1;
								} else {
									response.data['pureslaveNodes'].push(node);
									response.data['pureslaveCount'] += 1;
								}
							}

							response.data.instances += response.data['masterCount'];
							response.data.instances += response.data['sharedCount'];
							response.data.instances += response.data['pureslaveCount'];

							// if(response.data.type=="customized"){
							// 	response.data.instances = response.data.createNode.nodes.length;
							// }else{
							// 	if(response.data.createCategory=='compact'){
							// 		response.data.instances += 2;
							// 	}else{
							// 		response.data.instances += 5;
							// 	}
							// };

							ClusterService.createCluster(response.data).then(
								function(data) {
									$scope.$storage.cluster = data.data;
									getClusters();
								},
								function(error) {
									ResponseService.errorResponse(error);
							});
						}
					});

			}, function(error) {
				ResponseService.errorResponse(error);
			});

		};
		$scope.setCluster = function() {
			ClusterService.getClusters('', '', 'unterminated', '').then(function(data) {
					$scope.clusters = data.data;
					$scope.deleteClusters = _.filter($scope.clusters, function(cluster) {
						return cluster.status != 'TERMINATING' && cluster.status != 'INSTALLING' && cluster.status != 'MODIFYING';
					});
					if (_.find($scope.clusters, function(item) {
							return item._id == JSON.parse($scope.selectCluster)._id
						})) {
						$scope.$storage.cluster = _.find($scope.clusters, function(item) {
							return item._id == JSON.parse($scope.selectCluster)._id
						});
						$scope.selectCluster = JSON.stringify($scope.$storage.cluster);
					} else {
						$scope.selectCluster = $scope.clusters[0];
						$scope.$storage.cluster = JSON.parse($scope.selectCluster);
					}
				},
				function(errorMessage) {
					ResponseService.errorResponse(errorMessage);
				});
		}
		$scope.deleteCluster = function() {
			$uibModal.open({
					templateUrl: 'templates/cluster/deleteCluster.html',
					controller: 'deleteClusterController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								"clusters": $scope.deleteClusters
							}
						}
					}
				}).result
				.then(function(response) {
					if (response.operation === 'execute') {
						$scope.$translate(['common.deleteConfirm', 'node.deleteNodeMessage', 'common.delete', 'node.failedNode']).then(function(translations) {
							$scope.confirm = {
								"title": translations['common.deleteConfirm'],
								"message": {
									"main": translations['node.deleteNodeMessage'],
									"name": response.name,
									"default": translations['node.failedNode']
								},
								"button": {
									"text": translations['common.delete'],
									"action": function() {
										ClusterService.terminateCluster(response.data).then(function(data) {
												ClusterService.getClusters('', '', 'unterminated', '').then(function(data) {
														$scope.clusters = data.data;
														$scope.deleteClusters = _.filter($scope.clusters, function(cluster) {
															return cluster.status != 'TERMINATING' && cluster.status != 'INSTALLING' && cluster.status != 'MODIFYING';
														});
														if ($scope.clusters[0]) {
															$scope.$storage.cluster = $scope.clusters[0];
														} else {
															$scope.$storage.cluster = {};
														}
														$scope.selectCluster = JSON.stringify($scope.$storage.cluster);
													},
													function(errorMessage) {
														ResponseService.errorResponse(errorMessage);
													});
											},
											function(error) {
												ResponseService.errorResponse(error);
											});
									}
								}
							};
							ClusterService.deleteConfirm($scope);
						});
					}
				});
		};

		$scope.toggleCMI = function() {
			ClusterService.getClusterSetting( $scope.$storage.cluster._id, !$scope.$storage.cluster.setProjectvalue.cmi ).then(function( data ) {
				console.log( data );
				$uibModal.open({
					templateUrl: 'templates/common/success.html',
					controller: 'SuccessController',
					resolve: {
						model: function() {
							return {
								title: 'common.success',
								message: 'cluster.toggleCMISuccess',
								button: {
									text: 'common.ok'
								}
							}
						}
					}
				})
				.result
				.then(function( res ) {
					$scope.$storage.cluster.setProjectvalue.cmi = !$scope.$storage.cluster.setProjectvalue.cmi;
					window.location.reload();
				});
			}, function(error) {
				ResponseService.errorResponse(error);
			});
		};

		$scope.init = function() {
			getClusters();
			UserService.getUserInfo()
			.then(function (data) {
				$scope.user = data.data;
				// console.log('User owner info: ');
				// console.log($scope.user);
			})
		};
		$scope.init();
	}]);
	app.controllerProvider.register('createClusterController', ['$scope', '$uibModalInstance', 'model', 'ClusterService',
		function($scope, $uibModalInstance, model, ClusterService) {
			$scope.status={
				'open':false
			}
			$scope.validate = {
				"fromUser": true,
				"fromCluster": true
			};
			$scope.ClusterTypes = [{
				value: 'openstack'
			}, {
				value: 'amazonec2'
			}, {
				value: 'google'
			}, {
				value: 'customized'
			}];
			$scope.Providers = model.providers;
			$scope.LoginKeys = model.keypairs;
			$scope.Registries = model.registries;

			$scope.closeRegistrySelect=function(){
				$scope.RegistryIsOpen=false;
			}
			$scope.pushRegistry = function(index) {
				var temp=$scope.Registries.splice(index, 1);
				$scope.cluster.dockerRegistries.push(temp[0]);
				$scope.closeRegistrySelect();
			};
			$scope.popRegistry = function(index) {
				var temp=$scope.cluster.dockerRegistries.splice(index, 1);
				$scope.Registries.push(temp[0]);
				$scope.closeRegistrySelect();
			};
			$scope.pushLabel = function() {
				$scope.cluster.engineOpts.push({});
			};
			$scope.popLabel = function(index) {
				$scope.cluster.engineOpts.splice(index, 1);
			};

      var getInitNodes = function (type) {
        if(type === "compact") {
          // as a lite, reduce to 1 master 1 slave
          return [{
            ip: "", sshUser: "", type: "master", fixed: true
          },{
            ip: "", sshUser: "", type: "share", fixed: true
          }];
        }

        return [{
          ip: "", sshUser: "", type: "master", fixed: true
        }, {
          ip: "", sshUser: "", type: "master", fixed: true
        }, {
          ip: "", sshUser: "", type: "master", fixed: true
        }, {
          ip: "", sshUser: "", type: "share", fixed: true
        }, {
          ip: "", sshUser: "", type: "share",fixed: true
        }];
      };

      var init = function (type) {
        $scope.cluster = {
          type: type, name: '', owner: "", details: "",
          createCategory: "compact", nodeAttribute: "",
          pubkeyId: "", instances: 0, masterCount: 0,
          sharedCount: undefined, pureslaveCount: undefined,
          providerId: $scope.Providers.length ? $scope.Providers[0]._id : "",
          dockerRegistries:[], engineOpts:[],
          createNode: {
            privateKey: "",
            privateNicName: "", 
            nodes: getInitNodes("compact")
          }
        };
      };

      $scope.$watch('cluster.createCategory', function (nextVal) {
        $scope.cluster.createNode.nodes = getInitNodes(nextVal)
      });

      $scope.$watch('cluster.type', function(nextVal) {
        // Always initialize wehn the type is changed
        init(nextVal);

        if (nextVal == 'customized' && $scope.LoginKeys.length) {
          $scope.cluster.pubkeyId = "";
        }

        $scope.status.open = false;
        $scope.Providers = _.filter(model.providers, { type: nextVal });
        $scope.cluster.providerId = $scope.Providers.length ? $scope.Providers[0]._id : "";
			});
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.cluster
				});
			};
			$scope.popCluster = function(index) {
				$scope.cluster.createNode.nodes.splice(index, 1);
			};
      $scope.pushCluster = function() {
        $scope.cluster.createNode.nodes.push({
          ip: "", sshUser: "", type: "share", fixed: false
        });
      };
			$scope.validateClusterForUser = function(from) {
				var clustername = $scope.cluster.name;
				var pattern = /^[A-Za-z0-9\-]+$/;
				if (!angular.isUndefined(clustername) && !_.isEmpty(clustername) &&
					pattern.test(clustername)) {
					ClusterService.validateClusterForUser(clustername).then(function(data) {
							$scope.validate = {
								"fromUser": true,
								"fromCluster": true
							};
						},
						function(errorMessage) {
							if (from == "user") {
								$scope.validate = {
									"fromUser": false,
									"fromCluster": true
								};
							} else {
								$scope.validate = {
									"fromUser": true,
									"fromCluster": false
								};
							}
						});
				} else {
					$scope.validate = {
						"fromUser": true,
						"fromCluster": true
					};
				}
			};

      init($scope.ClusterTypes[0].value);
		}
	]);
	app.controllerProvider.register('deleteClusterController', ['$scope', '$uibModalInstance', 'model', '$localStorage',
		function($scope, $uibModalInstance, model, $localStorage) {
			$scope.$storage = $localStorage;
			$scope.clusters = model.clusters;
			if ($scope.$storage.cluster.status != 'TERMINATING' && $scope.$storage.cluster.status != 'INSTALLING' && $scope.$storage.cluster.status != 'MODIFYING') {
				$scope.selectedCluster = $scope.$storage.cluster._id;
			} else {
				$scope.selectedCluster = $scope.clusters[0]._id;
			}
			$scope.$watch('selectedCluster',function(){
				$scope.name=_.find($scope.clusters,function(item){
					return item._id==$scope.selectedCluster
				})
			})
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.selectedCluster,
					"name": $scope.name.name
				});
			};
		}
	]);

});
