define(['app', 'services/utils/common', 'services/utils/response', 'services/framework/main', 'directives/tree'], function(app) {
	'use strict';
	app.controllerProvider.register('FrameworkDetailController', ['$scope', '$localStorage', '$state', '$uibModal', 'CommonService', 'ResponseService', 'FrameworkService', function($scope, $localStorage, $state, $uibModal, CommonService, ResponseService, FrameworkService) {
		$scope.$storage = $localStorage;
		
		$scope.getDetail = function() {
			if(CommonService.endPointAvailable()) {
				FrameworkService.getDescribe(CommonService.getEndPoint(), 'describe', $state.params.frameworkName, $state.params.packageVersion).then(function(data) {
					$scope.description = data.package;
					$scope.images = data.resource.images;
					$scope.config = data.config;
					$scope.primaryInstallIsDisabled = FrameworkService.isDisabled($scope.config['properties']);
					$scope.options = FrameworkService.changeJson($scope.config['properties']);
				}, function(error) {
					ResponseService.errorResponse(error, "framework.frameworkDescriptionFailed");
				})
			}
		};
		
		$scope.$watch('$storage.cluster', function(newValue, oldValue) {
			if(newValue._id !== oldValue._id) {
				$state.go('framework');
			}
		}, true);

		$scope.primaryInstall = function() {
			FrameworkService.install(CommonService.getEndPoint(), $scope.description.name, $scope.options, $scope.description.version).then(function(data) {
				$scope.$translate(['framework.actionSuccess', 'framework.installSuccess']).then(function(translations) {
					$scope.prompt = {
						"title": translations['framework.actionSuccess'],
						"prompt_message": $scope.description['postInstallNotes'],
						"content": translations['framework.installSuccess']
					}
					FrameworkService.prompt($scope);
				});
			}, function(error) {
				if(error['code'] === 400) {
					error['errormsg'] = error['data']['message'];
				}
				ResponseService.errorResponse(error);
			});
		}
		
		$scope.beforePrimaryInstall = function() {
			$scope.$translate(['framework.installConfirm', 'framework.installMessage', 'framework.oneClickInstall']).then(function(translations) {
				$scope.prompt = {
					"title": translations['framework.installConfirm'],
					"prompt_message": $scope.description['preInstallNotes'],
					"content": translations['framework.installMessage'],
					"button": {
						"text": translations['framework.oneClickInstall'],
						"action": function() {
							$scope.primaryInstall();
						}
					}
				}
				FrameworkService.prompt($scope);
			});
		}

		$scope.advancedInstallType = function() {
			$uibModal.open({
				templateUrl: 'templates/framework/advancedInstall.html',
				controller: 'AdvancedInstallController',
				backdrop: 'static',
				size: 'lg',
				resolve: {
					model: function() {
						return {
							name: $scope.description.name,
							version: $scope.description.version,
							image: $scope.images ? $scope.images['icon-medium'] : '',
							config: $scope.config,
							preInstallNotes: $scope.description['preInstallNotes'],
							postInstallNotes: $scope.description['postInstallNotes']
						}
					}
				}
			});
		}

		$scope.getDetail();
	}]);
	
	app.controllerProvider.register('AdvancedInstallController', ['$scope', '$uibModal', '$uibModalInstance', 'model', 'FrameworkService', 'CommonService', 'ResponseService', function($scope, $uibModal, $uibModalInstance, model, FrameworkService, CommonService, ResponseService) {
		$scope.name = model.name;
		$scope.version = model.version;
		$scope.image = model.image;
		$scope.config = model.config;
		$scope.properties = model.config['properties'];
		$scope.showJson = FrameworkService.getShowJson($scope.properties);
	
	    $scope.setShowList = function(name) {
	       	_.each($scope.showJson, function(value, key, list) {
		        list[key] = false;
		        if(key == name) {
		          list[key] = true;
		        }
		    });
		}

	    	$scope.beforeAdvancedInstall = function() {
	    		$uibModalInstance.close('close');
			$scope.$translate(['framework.installConfirm', 'framework.installMessage', 'framework.advancedInstall']).then(function(translations) {
				$scope.prompt = {
					"title": translations['framework.installConfirm'],
					"prompt_message": model['preInstallNotes'],
					"content": translations['framework.installMessage'],
					"button": {
						"text": translations['framework.advancedInstall'],
						"action": function() {
							$scope.submit();
						}
					}
				}
				FrameworkService.prompt($scope);
			});
		}
		
		$scope.submit = function() {
		    //将$scope.properties转化成规定的格式
		    var options = FrameworkService.changeJson($scope.properties);
		    
		    FrameworkService.advancedInstall(CommonService.getEndPoint(), $scope.name, options, $scope.version).then(function(data) {
				$scope.$translate(['framework.actionSuccess', 'framework.installSuccess']).then(function(translations) {
					$scope.prompt = {
						"title": translations['framework.actionSuccess'],
						"prompt_message": model['postInstallNotes'],
						"content": translations['framework.installSuccess']
					}
					FrameworkService.prompt($scope);
				});
		    }, function(error) {
		    		if(error['code'] === 400) {
					error['errormsg'] = error['data']['message'];
				}
		    		ResponseService.errorResponse(error);
		    });
		}
		
		$scope.close = function(res) {
			$uibModalInstance.close(res);
		}
		
		
	}])
})
