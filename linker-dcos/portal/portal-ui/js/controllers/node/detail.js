define(['app', 'services/node/main','services/framework/main','services/configuration/platform'], function(app) {
	'use strict';
	app
		.controllerProvider
		.register('NodeDetailController', ['$scope', '$location','$localStorage','$state','$stateParams', 'CommonService', 'NodeService', 'ResponseService', 'FrameworkService','PlatformService',function($scope, $location,$localStorage,$state,$stateParams, CommonService, NodeService, ResponseService,FrameworkService,PlatformService) {
			$scope.$storage = $localStorage;
			if ($scope.$storage.node && $stateParams.nodeid == $scope.$storage.node.hostId) {
				$scope.nodeInfo = $scope.$storage.node;
			}else{
				$state.go('node');
			};
			$scope.confirmDelete = function() {
				$scope.$translate(['common.deleteConfirm', 'node.deleteMessage', 'common.delete']).then(function(translations) {
					$scope.confirm = {
						"title": translations['common.deleteConfirm'],
						"message": translations['node.deleteMessage'],
						"button": {
							"text": translations['common.delete'],
							"action": function() {
								$scope.deleteNode();
							}
						}

					};
					CommonService.deleteConfirm($scope);
				});
			};
			$scope.getData = function(type) {
				var skip = ($scope.checkdata.currentPage - 1) * $scope.recordPerPage;
				var limit = $scope.recordPerPage;
				if(CommonService.endPointAvailable()){
					if(type=='container'){
						NodeService.getContainers(CommonService.getEndPoint(),$scope.nodeInfo.privateIp,skip,limit).then(function(data){
							$scope.containers = data.data;
							$scope.totalrecords = data.count;
							$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
						},function(errorMessage) {
							ResponseService.errorResponse(errorMessage);
						});
					}else{
						FrameworkService.getTasks(CommonService.getEndPoint(),$scope.nodeInfo.privateIp,skip,limit).then(function(data){
							$scope.tasks = data.data;
							$scope.totalrecords = data.count;
							$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
						},function(errorMessage) {
							ResponseService.errorResponse(errorMessage);
						});
					}
				}
			};
			$scope.deleteNode = function() {
				NodeService.terminateNode({
					_id: [$scope.nodeInfo.hostId],
					cluster_id: $scope.nodeInfo.clusterId
				}).then(function(data) {
					$scope.$storage.node={};
					$state.go('node');
				}, function(errorMessage) {
					ResponseService.errorResponse(errorMessage);
				});
			}
			var init = function(type) {
				$scope.totalPage = '';
				$scope.recordPerPage = CommonService.recordNumPerPage();
				$scope.totalrecords = '';
				$scope.checkdata={
					currentPage:1,
					type:type
				}
			};
			init('container');
			if ($scope.$storage.cluster.providerId && $scope.$storage.node.type!='customized') {
				PlatformService.queryProvider($scope.$storage.cluster.providerId).then(
					function(data) {
						$scope.provider_name = data.data.name;
					},
					function(errorMessage) {
						ResponseService.errorResponse(errorMessage);
					}
				);
			}
			$scope.$watch('checkdata', function(nv, ov) {
					$scope.getData($scope.checkdata.type);
			},true);
			$scope.$watch('$storage.cluster',function(nv,ov){
				if(nv==ov)return;
				$state.go('node');
			},true)
			$scope.changeType=function(type){
				init(type)
			};
			$scope.goToMonitoring = function(){
				$localStorage.selectedNodeToMonitor = $scope.nodeInfo.privateIp;
				$location.path("/monitor");
			}
		}]);
});