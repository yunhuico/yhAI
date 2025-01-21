define(['app', 'services/framework/main', 'directives/pagination', 'directives/status'], function(app, FrameworkService) {
    'use strict';
    app.controllerProvider.register('FrameworkController', ['$scope', '$uibModal', '$localStorage', '$location', 'FrameworkService', 'CommonService', 'ResponseService', function($scope, $uibModal, $localStorage, $location, FrameworkService, CommonService, ResponseService) {

        $scope.$storage = $localStorage;
		$scope.query = {};
		var status = {
			category: true,
			tasks: false,
			repository: false
		};
		var setStatus = function(type) {
			_.each(status, function(val, key) {
				if(key == type) {
					status[key] = true;
				}
			});
		};
		var initStatus = function() {
			_.each(status, function(val, key) {
				status[key] = false;
			});
		};
        $scope.endPointAvailable = CommonService.endPointAvailable();
        $scope.$watch('$storage.cluster', function(newValue, oldValue) {
            if (!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id) {
                $scope.endPointAvailable = CommonService.endPointAvailable();
                $scope.frameworks = [];
               _.each(status, function(val, key) {
               		if(key == 'category' && val == true) {
               			$scope.query.name = '';
                			$scope.getFrameworks($scope.query.name);
               		}else if(key == 'tasks' && val == true) {
               			$scope.getInstalledFrameworks();
               		}else if(key == 'repository' && val == true) {
               			$scope.getRepository();
               		}
               })
            }

        }, true);

		$scope.getFrameworks = function(name) {
			initStatus();
			setStatus('category');
			if ($scope.endPointAvailable) {
				if(_.isUndefined(name) || _.isEmpty(name)) {
					name = '';
				}
                FrameworkService.get(CommonService.getEndPoint(), 'search', name).then(function(data) {
                        $scope.frameworks = data.packages;
                    },
                    function(error) {
                    		if(error['code'] === 400) {
							error['errormsg'] = error['data']['message'];
						}
                    		$scope.frameworks = [];
                    		ResponseService.errorResponse(error);
                    })
            } else {
                $scope.frameworks = [];
            }
		};

		$scope.promptAlert = function(data) {
			$scope.$translate(['framework.actionSuccess', 'framework.uninstallSuccess']).then(function(translations) {
				$scope.prompt = {
					"title": translations['framework.actionSuccess'],
					"prompt_message": data[0]['postUninstallNotes'],
					"content": translations['framework.uninstallSuccess']
				}
				FrameworkService.prompt($scope);
			});
		}

        $scope.getInstalledFrameworks = function() {
        		initStatus();
        		setStatus('tasks');
            if ($scope.endPointAvailable) {
                FrameworkService.get(CommonService.getEndPoint(), 'list').then(function(data) {
                        $scope.installedFrameworks = data.packages;
                        if($scope.isPrompt) {
                        		$scope.promptAlert($scope.isPrompt);
                        		$scope.isPrompt = '';
                        }
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "framework.listFailed");
                    })
            } else {
                $scope.installedFrameworks = [];
            }
        };

		$scope.confirmUninstall = function(item) {
			$scope.$translate(['framework.uninstallConfirm', 'framework.uninstallMessage', 'common.uninstall']).then(function(translations) {
				$scope.confirm = {
					"title":translations['framework.uninstallConfirm'],
					"message": translations['framework.uninstallMessage'],
					"button": {
						"text": translations['common.uninstall'],
						"action": function() {
							$scope.uninstallFramework(item);
						}
					}
				}
				CommonService.deleteConfirm($scope);
			})
		};

		$scope.uninstallFramework = function(item) {
			FrameworkService.uninstall(CommonService.getEndPoint(), item.packageInformation.packageDefinition.name, $scope.installedFrameworks).then(function(data) {
				$scope.isPrompt = data.results;
				$scope.getInstalledFrameworks();
			}, function(error) {
				ResponseService.errorResponse(error, "framework.uninstallFailed");
			});
		};

		$scope.getRepository = function() {
			initStatus();
			setStatus('repository');
			if($scope.endPointAvailable) {
				FrameworkService.getRepository(CommonService.getEndPoint()).then(function(data) {
					$scope.repos = data.repositories;
				},
				function(error) {
					ResponseService.errorResponse(error, "framework.getRepositoryFailed");
				})
			}else {
				$scope.repos = [];
			}
		}

		$scope.createRepository = function() {
			$uibModal.open({
				templateUrl:'templates/framework/createRepository.html',
				controller: 'CreateRepositoryController',
				backdrop:'static',
			})
			.result
			.then(function(response) {
				if (response.operation === 'execute') {
					FrameworkService.createRepository(CommonService.getEndPoint(), response.data).then(function(data) {
						$scope.getRepository()
					},
					function(error) {
						ResponseService.errorResponse(error);
					});
				}
			})
		}

		$scope.confirmDeleteRepo = function(repo) {
			$scope.$translate(['common.deleteConfirm', 'framework.deleteRepository', 'common.delete']).then(function(translations) {
				$scope.confirm = {
					"title":translations['common.deleteConfirm'],
					"message": translations['framework.deleteRepository'],
					"button": {
						"text": translations['common.delete'],
						"action": function() {
							$scope.deleteRepository(repo);
						}
					}
				}
				CommonService.deleteConfirm($scope);
			})
		}

		$scope.deleteRepository = function(repo) {
			FrameworkService.deleteRepository(CommonService.getEndPoint(), repo).then(function(data) {
				$scope.getRepository();
			}, function(error) {
				ResponseService.errorResponse(error);
			})
		}
    }]);


	app.controllerProvider.register('CreateRepositoryController', ['$scope', '$uibModalInstance', function($scope, $uibModalInstance) {
		$scope.repository = {
			"name": "",
			"uri": ""
		}
		$scope.close = function(res) {
			$uibModalInstance.close({
				"operation": res,
				"data": $scope.repository
			});
		}
	}]);

});
