define(['app', 'services/node/main', 'services/cluster/main','services/configuration/platform','directives/pagination', 'directives/status', 'services/configuration/dockerregistries', 'services/configuration/key'], function(app) {
	'use strict';
	app
		.controllerProvider
		.register('NodeController', ['$scope', '$state', '$localStorage', '$stateParams', '$location', '$uibModal', 'NodeService','PlatformService', 'ResponseService', 'CommonService', 'ClusterService', 'DockerregistriesService', 'KeyPairService', function($scope, $state, $localStorage, $stateParams, $location, $uibModal, NodeService,PlatformService, ResponseService, CommonService, ClusterService, DockerregistriesService, KeyPairService) {
			$scope.$storage = $localStorage;
			$scope.nodedetail = function(detail) {
				if(detail.status!="OFFLINE"&&detail.status!="FAILED"){
					$scope.$storage.node = detail;
					$state.go("nodedetail", {
						nodeid: detail.hostId
					});
				}
			};
			$scope.refreshCluster = function() {
				ClusterService.getCluster($scope.$storage.cluster._id).then(function(data) {
					if ($scope.$storage.cluster._id == data.data._id) {
						if ($scope.$storage.cluster.status == data.data.status) {
							$scope.getNode()
						} else {
							if (data.data.status != "TERMINATED") {
								$scope.$storage.cluster = data.data;
								$scope.$parent.selectCluster = JSON.stringify($scope.$storage.cluster);
								_.find($scope.$parent.clusters, function(item) {
									if (item._id == $scope.$storage.cluster._id) {
										angular.copy($scope.$storage.cluster,item);
									}
								})
								$scope.$parent.deleteClusters = _.filter($scope.$parent.clusters, function(cluster) {
									return cluster.status != 'TERMINATING' && cluster.status != 'INSTALLING' && cluster.status != 'MODIFYING';
								});
							} else {
								ClusterService.getClusters('', '', 'unterminated', '').then(function(data) {
										$scope.$parent.clusters = data.data;
										$scope.$parent.deleteClusters = _.filter($scope.$parent.clusters, function(cluster) {
											return cluster.status != 'TERMINATING' && cluster.status != 'INSTALLING' && cluster.status != 'MODIFYING';
										});
										if ($scope.$parent.clusters[0]) {
											$scope.$storage.cluster = $scope.$parent.clusters[0];
										} else {
											$scope.$storage.cluster = {};
										}
										$scope.$parent.selectCluster = JSON.stringify($scope.$storage.cluster);
									},
									function(errorMessage) {
										ResponseService.errorResponse(errorMessage);
									});
							}
						}
					} else {
						_.find($scope.$parent.clusters, function(item) {
							if (item._id == data.data._id) {
								item.status = data.data.status
							}
						})
					}
				}, function(errorMessage) {
					ResponseService.errorResponse(errorMessage);
				})
			}
			$scope.getNode = function() {
				if (CommonService.clusterAvailable()) {
					var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
					var limit = $scope.recordPerPage;
					NodeService.getNodes($scope.$storage.cluster._id, skip, limit).then(function(data) {
						if(data.config.url.split('/')[2]==$scope.$storage.cluster._id){
							$scope.totalrecords = data.data.length;
							$scope.totalPage = Math.ceil(data.count / $scope.recordPerPage);
							$scope.node = data.data;
							$scope.infoStatus=_.some($scope.node, function(item){
								if(item.status=='FAILED'){
									return true
								}else{
									return false
								}
							});
						}

						// setup filter flag
						_.each($scope.node, function (node, keyNode) {
							$scope.node[keyNode].show = true;
						});

						// @TODO tag (note), have to check if it migrate to production mode
						// how does it work? if works well, please remove this comment
						_.each($scope.node, function (node, keyNode) {
							$scope.node[keyNode].note = [];
							// @TODO: 先註解，CMI 功能開關需連動
							// NodeService.getTags(node.hostName)
							// .then(function (data) {

							// 	if ( ! data.data) {
							// 		return;
							// 	}
							// 	var result;
							// 	try {
							// 		result = JSON.parse(data.data);
							// 	} catch (e) {
							// 		result = data.data;
							// 	}
							// 	var note = _.map(result, 'tag_class');

							// 	$scope.node[keyNode].note = note;
							// 	// $scope.node[keyNode].note.push('cpu_low');

							// });
						});
					}, function(errorMessage) {
						ResponseService.errorResponse(errorMessage);
					});
				} else {
					$scope.node = [];
					$scope.totalrecords = 0;
					$scope.totalPage = 1;
				}
			};
			$scope.selectAll = function(checked) {
				_.each($scope.node, function(item) {
					if (!item.isMasterNode  && (item.status != 'TERMINATING') && (item.status != 'INSTALLING')) {
						item.select = checked;
					}
				})
			};
			$scope.$watch('node', function(nv, ov) {
				if (nv == ov) return;
				if (!$scope.node) {
					$scope.nodeSelectAll = false;
					return
				}
				if ($scope.node && $scope.node.length) {
					$scope.nodeSelectAll = _.every($scope.node, function(item) {
						if (!item.isMasterNode && (item.status != 'TERMINATING') && (item.status != 'INSTALLING')) {
							return item.select;
						}
            // Always takes non-selectable node as true
            return true;
					});
				} else {
					$scope.nodeSelectAll = false;
				}
			}, true);
			$scope.addNode = function() {
				$uibModal.open({
						templateUrl: 'templates/node/addNode.html',
						controller: 'createNodeController',
						backdrop: 'static',
						resolve: {
							model: function() {
								return {

								};
							}
						}
					})
					.result
					.then(function(response) {
						if (response.operation === 'execute') {
							_.each(response.data.label, function(item) {
								if (item.key && item.value) {
									response.data.nodeAttribute += item.key + ':' + item.value + ';';
								}
							});
							if (!_.isUndefined(response.data.addNode.privateKey)) {
								response.data.addNode.privateKey = Base64.encode(response.data.addNode.privateKey);
							}
							response.data.label = null;

							response.data['sharedCount'] = (response.data['sharedCount'] && response.data['sharedCount'] > 0) ? response.data['sharedCount'] : 0;
							response.data['pureslaveCount'] = (response.data['pureslaveCount'] && response.data['pureslaveCount'] > 0) ? response.data['pureslaveCount'] : 0;
							response.data['sharedNodes'] = [];
							response.data['pureslaveNodes'] = [];
							var otherNodes = response.data.addNode.nodes;

							for (var i in otherNodes) {
								var node = otherNodes[i];
								node['privateKey'] = response.data.addNode.privateKey;
								node['privateNicName'] = response.data.addNode.privateNicName;

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

							// Get total nodes
							if (response.data.addMode == 'reuse' || $scope.$storage.cluster.type == 'customized') {
								response.data.addNumber = response.data.addNode.nodes.length;
							}else{
								response.data.addNumber += response.data['sharedCount'];
								response.data.addNumber += response.data['pureslaveCount'];
								while(response.data.addNumber > response.data.addNode.nodes.length){
									response.data.addNode.nodes.push({ip: "",sshUser: ""})
								}
							}

							NodeService.createNode(response.data).then(function() {
								$scope.refreshCluster();
							}, function(errorMessage) {
								$scope.refreshCluster();
								ResponseService.errorResponse(errorMessage);
							})
						}
					});
			};

			$localStorage.nodeOptions = [{
				isUsed: false,
				type: 'silver',
				cpu: '1',
				mem: '3.75',
				// privateIp: '10.140.0.2'
				privateIp: '10.140.0.17',
				price: '$99',
				storage: '40GB'
			}, {
				isUsed: false,
				type: 'gold',
				cpu: '2',
				mem: '7.5',
				// privateIp: '10.140.0.3'
				privateIp: '10.140.0.16',
				price: '$199',
				storage: '120GB'
			}, {
				isUsed: false,
				type: 'platinum',
				cpu: '4',
				mem: '15',
				// privateIp: '10.140.0.4'
				privateIp: '10.140.0.15',
				price: '$399',
				storage: '250GB'
			}];

			$scope.updatePublickey = function() {
				$uibModal.open({
					templateUrl: 'templates/node/publickey.html',
					controller: 'updatePublickeyController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								nodeOptions: $scope.$storage.nodeOptions
							};
						}
					}
				})
				.result
				.then(function( result ) {
					if ( result.operation == 'execute' ) {
						KeyPairService.addPublickeyToCluster(result.data.add, $scope.$storage.cluster._id)
						.then(function (data) {
							KeyPairService.deletePublickeyToCluster(result.data.delete, $scope.$storage.cluster._id)
							.then(function (data) {
								$uibModal.open({
									templateUrl: 'templates/common/success.html',
									controller: 'SuccessController',
									resolve: {
										model: function() {
											return {
												title: 'common.success',
												message: 'node.updatePublickeySuccess',
												button: {
													text: 'common.close'
												}
											}
										}
									}
								})
								.result
								.then(function( res ) {
									window.location.reload();
								});
								// return alert("Update Public key success, page will reload");
							}, function( errorMessage ) {
								ResponseService.errorResponse(errorMessage);
							});
						}, function( errorMessage ) {
							ResponseService.errorResponse(errorMessage);
						});
					}
				});
			};

			$scope.changeNodeState = function() {
				$scope.getRunningFakeNode = function() {
					var sum = 0;
					$scope.node.forEach(function( node ) {
						if ( node.status == "RUNNING" )
							sum++;
					});

					return sum;
				}

				$scope.getStoppedFakeNode = function() {
					var sum = 0;
					$scope.node.forEach(function( node ) {
						if ( node.status == "STOPPED" )
							sum++;
					});

					return sum;
				}

				$uibModal.open({
					templateUrl: 'templates/node/changeNodeState.html',
					controller: 'changeNodeStateController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								type: 'START',
								maxStart: $scope.getStoppedFakeNode(),
								maxStop: $scope.getRunningFakeNode()
							};
						}
					}
				})
				.result
				.then(function( result ) {
					if ( result.operation == 'finish' ) {
						var nodeNum = result.data.nodeNum;
						if ( result.data.state == 'start' ) {
							$scope.node.forEach(function( node ) {
								if ( nodeNum > 0 ) {
									if ( node.status == "STOPPED" ) {
										nodeNum--;
										node.status = 'start';
										setTimeout(function() {
											node.status = "RUNNING";
											var noteTable = ['cpu low', 'mem low', 'cpu peak', 'mem peak'];
											node.note = [noteTable[Math.floor(Math.random() * (4 - 0)) + 0]];
											$scope.$apply();
										}, 10000);

										// console.log(Math.floor(Math.random() * (4 - 0)) + 0);
									}
								}
								else
									throw "complete, it's not an error";
							});
						} else if ( result.data.state == 'stop' ) {
							var note = result.data.note;
							$scope.node.forEach(function( node ) {
								if ( node.note && node.note.includes(note) ) {
									if ( node.status == "RUNNING" ) {
										nodeNum--;
										node.status = 'TERMINATING';
										setTimeout(function() {
											node.status = "STOPPED";
											node.note = null;
											$scope.$apply();
										}, 4000);
									}
								}
							});
						} // else if
					}
				});
			};

			$scope.terminateNode = function(item) {
				var node = {
					_id: [item.hostId],
					cluster_id: $scope.$storage.cluster._id
				}

				var feedback = {
					clusterid: $scope.$storage.cluster._id,
					hostNames: [item.hostName]
				};

				$scope.$translate(['common.nodeHasPipework', 'common.nodeHasPipework', 'common.deleteConfirm', 'node.deleteNodeMessage', 'common.delete', 'node.failedNode']).then(function(translations) {
					$scope.confirm = {
						"title": translations['common.deleteConfirm'],
						"message": {
							"main": translations['node.deleteNodeMessage'],
							"name": item.hostName,
							"default": translations['node.failedNode']
						},
						"button": {
							"text": translations['common.delete'],
							"action": function() {

								ClusterService.getNetworkOvs(CommonService.getEndPoint(), feedback)
								.then(function (data) {

									if (data.data && data.data.num > 0) {
                    ResponseService.errorResponse({
                      "code": 400,
                      "errormsg": translations['common.nodeHasPipework']
                    });
                  }

									NodeService.terminateNode(node).then(function() {
										$scope.refreshCluster();
									}, function(errorMessage) {
										$scope.refreshCluster();
										ResponseService.errorResponse(errorMessage);
									});
								})
								.catch(function (err) {
									console.log(err);
									$scope.refreshCluster();
									ResponseService.errorResponse(err);
								})
							}
						}

					};
					NodeService.deleteConfirm($scope);
				});
			};

			$scope.filterLoading = false;
			$scope.filterClick = function(opt) {

				$scope.filterLoading = false;

				var selectedOption = opt || $scope.filterSelect;

				// clear function
				if (selectedOption == "none") {
					_.each($scope.node, function (node, keyNode) {
						$scope.node[keyNode].show = true;
					});
					$scope.filterSelect = null;
					return;
				}

				$scope.filterLoading = true;
				// make all nodes disappear
				_.each($scope.node, function (node, keyNode) {
					$scope.node[keyNode].show = false;
				});

				NodeService.filterTag(selectedOption)
				.then(function (data) {
					$scope.filterLoading = false;

					if (data.data.length < 1) {
						return alert("No filter result");
					}

					var filterNodes = data.data;
					_.each(filterNodes, function (filterNode, idx) {
						_.each($scope.node, function (node, keyNode) {
							if (node.hostName == filterNode) {
								$scope.node[keyNode].show = true;
								return;
							}
						});
					});
				});
			};

			$scope.retrainClickHandler = function() {

				$scope.filterLoading = true;
				var filterItem = $scope.filterSelect || "slack";
				NodeService.retrain(filterItem)
				.then(function (data) {
					$scope.filterLoading = false;
					return alert("Retrain data upload.");
				});
			};

			$scope.feedbackClickHandler = function() {
				$scope.filterLoading = true;

				// /v1/cmi/nodes/feedback
				var nodefeedbacks = [];
				_.each($scope.node, function(node, key) {
					if ( ! node.feedback)
						return;

					nodefeedbacks.push({
						hostname: node.hostName,
						feedback: node.feedback
					});
				});

				var filterItem = $scope.filterSelect || "slack";

				var feedbackObj = {
					nodefeedbacks: nodefeedbacks,
					filteritem: filterItem
				};

				// Service
				NodeService.feedbackTags(feedbackObj)
				.then(function(data) {
					$scope.filterLoading = false;
					return alert('Feedback data upload.');
				});
			};

			$scope.updateRegistry = function() {
				$uibModal.open({
					templateUrl: 'templates/node/registry.html',
					controller: 'updateRegistryController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								nodeOptions: $scope.$storage.nodeOptions
							};
						}
					}
				})
				.result
				.then(function( result ) {
					if ( result.operation == 'execute' ) {
						DockerregistriesService.addDockerregistryToCluster(result.data.add, $scope.$storage.cluster._id)
						.then(function (data) {
							DockerregistriesService.deleteDockerregistryToCluster(result.data.delete, $scope.$storage.cluster._id)
							.then(function (data) {
								window.location.reload();
								return alert("Update registry success, page will reload");
							});
						});
					}
				});
			};

			$scope.getFilterNode = function (nodes, type) {
				var datas = _.chain(nodes).filter(function (item) {
					if (!item.isMasterNode && (item.status != 'TERMINATING') && (item.status != 'INSTALLING')) {
						if (item.select == true) {
							return true;
						}
					}
					return false;
				}).map(function (item) { return item[type];}).value();

				return datas;
			};

			$scope.terminateNodes = function() {
				$scope.$translate(['common.deleteConfirm', 'node.deleteNodesMessage', 'common.delete']).then(function(translations) {
					$scope.confirm = {
						"title": translations['common.deleteConfirm'],
						"message": translations['node.deleteNodesMessage'],
						"button": {
							"text": translations['common.delete'],
							"action": function() {

								// check OVS nodes
								var hostNames = $scope.getFilterNode($scope.node, 'hostName');
								var feedback = {
									clusterid: $scope.$storage.cluster._id,
									hostNames: hostNames
								};
								
								// Delete nodes
								var nodes = {
									_id: [],
									cluster_id: $scope.$storage.cluster._id
								}
								nodes._id = $scope.getFilterNode($scope.node, 'hostId', true);

								if (nodes._id && nodes._id.length > 0) {
									ClusterService.getNetworkOvs(CommonService.getEndPoint(), feedback)
									.then(function (data) {
										if (data.data.num > 1)
											return alert(translations['common.nodeHasPipework']);

										NodeService.terminateNode(nodes).then(function() {
											$scope.refreshCluster();
										}, function(errorMessage) {
											$scope.refreshCluster();
											ResponseService.errorResponse(errorMessage);
										});
									})
									.catch(function (err) {
										console.log(err);
									})
										
								} else {
									$uibModal.open({
										templateUrl: 'templates/common/ConfirmAbsort.html',
										controller: 'ConfirmAbsortController',
										backdrop: 'static',
										resolve: {
											model: function() {
												return {
													message: {
														"title": "common.NoChoice",
														"content": "common.PleaseSelectTheCorrespondingNode",
														"button": "common.confirm"
													}
												};
											}
										}
									});
								}

							}
						}

					};
					CommonService.deleteConfirm($scope);
				});
			}

			$scope.showNodeInfo = function( nodeInfo, index ) {
				var that = $('#node-' + index);
				that.removeClass( 'transition' );
				that.addClass( 'highlight' );
				$(".right-content.ng-scope").animate({
					scrollTop: that.offset().top
				}, 500, function() {
					that.addClass( 'transition' );
					that.removeClass( 'highlight' );
				});
			};

			$scope.scrollTop = function() {
				$(".right-content.ng-scope").animate({
					scrollTop: 0
				}, 500);
			}

			var init = function() {
				//得到选择的id
				$scope.currentPage = 1;
				$scope.totalPage = 1;
				$scope.recordPerPage = CommonService.recordNumPerPage();
				$scope.totalrecords = 0;
				$scope.$watch('$storage.cluster', function() {
					$scope.currentPage = 1;
					$scope.totalPage = 1;
					$scope.totalRecords = 0;
					$scope.node = undefined;
					if ($scope.$storage.cluster && $scope.$storage.cluster.providerId) {
						PlatformService.queryProvider($scope.$storage.cluster.providerId).then(
							function(data) {
								$scope.provider_name = data.data.name;
							},
							function(errorMessage) {
								$scope.provider_name='-'
								ResponseService.errorResponse(errorMessage);
							}
						);
					}else{
						$scope.provider_name=''
					}
					$scope.getNode();
				}, true);
				$scope.$watch('currentPage', function(nv, ov) {
					if (nv == ov) return;
					$scope.node = undefined;
          $scope.getNode();
				});

				// add new slack and attacked name.
				$scope.filterOptions = [
					{ value: "slack", name: "Low Performance"},
					{ value: "attacked", name: "Threat and Risk"}
				];

			};
			init();
		}]);
	app.controllerProvider.register('ConfirmAbsortController', ['$scope', '$uibModalInstance', 'model',
		function($scope, $uibModalInstance, model) {
			$scope.message = model.message;
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res
				});
			};
		}
	]);
	app.controllerProvider.register('createNodeController', ['$scope', '$localStorage', '$uibModalInstance', 'model',
		function($scope, $localStorage, $uibModalInstance, model) {
			$scope.$storage = $localStorage;
			// $scope.$watch('status.open', function(nv, ov){
			//     $scope.node.engineOpts=[]
			// });
			$scope.status={
				'open':false
			}
			$scope.pushDocker = function() {
				$scope.node.engineOpts.push({});
			};
			$scope.popDocker = function(index) {
				$scope.node.engineOpts.splice(index, 1);
			};
			var init = function() {
				if ($scope.$storage.cluster.type != 'customized') {
					$scope.node = {
						addMode: "new",
						addNumber: 0,
						sharedCount: 0,
						pureslaveCount: 0,
						clusterId: $scope.$storage.cluster._id,
						nodeAttribute: "",
						label: [],
						addNode: {
							privateKey: "",
							nodes: []
						},
						engineOpts:[]
					};
					$scope.$watch('node.addMode', function(nv, ov) {
						if (nv == ov) return;
						if (nv == "new") {
							$scope.node.label = [];
							$scope.node.addNode = {
								privateNicName:"",
								privateKey: "",
								nodes: [{
									ip: "",
									sshUser: "",
									type: "share"
								}]
							};
						} else {
							$scope.node.label = [];
							$scope.node.addNode = {
								privateNicName:"",
								privateKey: "",
								nodes: [{
									ip: "",
									sshUser: "",
									type: "share"
								}]
							};
						}
						$scope.node.engineOpts=[];
						$scope.node.addNumber = 1;
						$scope.status.open=false
					});
				} else {
					$scope.node = {
						addMode: "reuse",
						addNumber: 1,
						clusterId: $scope.$storage.cluster._id,
						nodeAttribute: "",
						label: [],
						addNode: {
							privateNicName:"",
							privateKey: "",
							nodes: [{
								ip: "",
								sshUser: "",
								type: "share"
							}]
						},
						engineOpts:[]
					};
				}
			}
			$scope.pushLabel = function() {
				$scope.node.label.push({});
			};
			$scope.popLabel = function(index) {
				$scope.node.label.splice(index, 1);
			};
			$scope.popNode = function(index) {
				$scope.node.addNode.nodes.splice(index, 1);
			};
			$scope.pushNode = function() {
				$scope.node.addNode.nodes.push({
					ip: "",
					sshUser: "",
					type: "share"
				});
			};
			init();

			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.node
				});
			};
		}
	]);
	app.controllerProvider.register('changeNodeStateController', ['$scope', '$localStorage', '$uibModalInstance', '$location', '$uibModal', 'model',
		function($scope, $localStorage, $uibModalInstance, $location, $uibModal, model) {
			$scope.maxStart = model.maxStart || 0;
			$scope.maxStop = model.maxStop || 0;
			$scope.node = {
				nodeStatus: 'start',
				nodeNum: 0
			};
			$scope.tags = [{
				name: 'cpu low'
			}, {
				name: 'mem low'
			}, {
				name: 'mem peak'
			}, {
				name: 'cpu peak'
			}, {
				name: 'cpu_low'
			}];
			$scope.node.tag = 'cpu low';

			$scope.close = function(res) {
				if ( res == 'finish' ) {

					$uibModalInstance.close({
						"operation": res,
						data: {
							state: $scope.node.nodeStatus,
							nodeNum: $scope.node.nodeNum,
							note: $scope.node.tag
						}
					});
				}
				else
					$uibModalInstance.close({
						"operation": res
					});
			};
		}
	]);
	app.controllerProvider.register('updatePublickeyController', ['$scope', '$localStorage', '$uibModalInstance', 'model', 'KeyPairService',
		function($scope, $localStorage, $uibModalInstance, model, KeyPairService) {
			$scope.$storage = $localStorage;
			// $scope.$storage.cluster._id

			$scope.init = function () {
				KeyPairService.getKeyPairs()
				.then(function(data) {
					$scope.publicKeys = data.data;

					var _clusterPubkeyIds = $scope.$storage.cluster.pubkeyId;
					if (_clusterPubkeyIds.length <= 0)
						return;

					for (var i in $scope.publicKeys) {
						var key = $scope.publicKeys[i];
						// cluster key has no _id attribute
						var find = _.findIndex(_clusterPubkeyIds, function (item) {
							return (key['_id'] === item)
						});

						// find the key from array
						if (find > -1) {
							$scope.publicKeys[i].checked = true;
						}
					}

				});
			};

			$scope.init();

			$scope.toggleSelection = function (item) {
				var key = _.findIndex($scope.publicKeys, { '_id': item['_id']});
				$scope.publicKeys[key].checked = ! item.checked;

				console.log($scope.publicKeys);
			}

			$scope.close = function(res) {

				var returnPublicKey = {
					add: {
						"ids": []
					},
					delete: {
						"ids": []
					}
				};
				returnPublicKey.add.ids = _.filter($scope.publicKeys, function (item) {
					return item.checked;
				}).map(function (item) {
					return item._id;
				});

				returnPublicKey.delete.ids = _.filter($scope.publicKeys, function (item) {
					return ! item.checked;
				}).map(function (item) {
					return item._id;
				});

				$uibModalInstance.close({
					"operation": res,
					"data": returnPublicKey
				});
			};

		}
	]);
	app.controllerProvider.register('updateRegistryController', ['$scope', '$localStorage', '$uibModalInstance', 'model', 'DockerregistriesService',
		function($scope, $localStorage, $uibModalInstance, model, DockerregistriesService) {
			$scope.$storage = $localStorage;
			// $scope.$storage.cluster._id
			$scope.init = function () {
				DockerregistriesService.getDockerregistries('','')
				.then(function (data) {
					// leave secure item for user add to cluster
					$scope.registery = _.filter(data.data, function (item) {
						return item.secure;
					});

					var _clusterRegistries = $scope.$storage.cluster.dockerRegistries;

					for (var i in $scope.registery) {
						var registry = $scope.registery[i];
						var find = _.find(_clusterRegistries, { '_id': registry['_id']});
						if (find) {
							$scope.registery[i].checked = true;
						}
					}
				});
			};

			$scope.init();


			$scope.toggleSelection = function (item) {
				var key = _.findIndex($scope.registery, { '_id': item['_id']});
				$scope.registery[key].checked = ! item.checked;

				console.log($scope.registery);
			}

			$scope.close = function(res) {

				var returnRegistry = {
					add: {
						"ids": []
					},
					delete: {
						"ids": []
					}
				};
				returnRegistry.add.ids = _.filter($scope.registery, function (item) {
					return item.checked;
				}).map(function (item) {
					return item._id;
				});

				returnRegistry.delete.ids = _.filter($scope.registery, function (item) {
					return ! item.checked;
				}).map(function (item) {
					return item._id;
				});

				$uibModalInstance.close({
					"operation": res,
					"data": returnRegistry
				});
			};

		}
	]);
});
