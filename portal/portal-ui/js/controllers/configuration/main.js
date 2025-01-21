define(['app', 'services/configuration/key', 'services/configuration/smtp', 'services/configuration/platform', 'services/configuration/dockerregistries', 'services/user/main', 'directives/pagination'], function(app) {
	'use strict';
	app.controllerProvider.register('ConfigController', ['$scope', '$uibModal', '$location', '$state', '$cookies', 'KeyPairService', 'PlatformService', 'SMTPService', 'DockerregistriesService', 'UserService', 'CommonService', 'ResponseService', function($scope, $uibModal, $location, $state, $cookies, KeyPairService, PlatformService, SMTPService, DockerregistriesService, UserService, CommonService, ResponseService) {
		(function() {
      UserService.getUserInfo()
      .then(function (data) {
        $scope.rolename = data.data.rolename;
      });
			$scope.tabcontrol = {
				"keypair": false,
				"smtp": false,
				"platform": false,
				"registry": false
			}
			$scope.$watch('$state.current.name', function(nv, ov) {
				_.each($scope.tabcontrol, function(val, key) {
					if (key == nv) {
						$scope.tabcontrol[key] = true
					} else {
						$scope.tabcontrol[key] = false
					}
				})
			});
			if ($state.current.name == 'keypair') {
				$scope.keypairs = [];
				$scope.users = [];
				$scope.$watch('currentPage', function() {
					$scope.getKeyPairs();
				});
				$scope.recordPerPage = 2;

			}
			if ($state.current.name == 'smtp') {
				$scope.smtpservers = [];
				$scope.$watch('currentPage', function() {
					$scope.getSMTPServers();
				});
				$scope.recordPerPage = CommonService.recordNumPerPage();
			}
			if ($state.current.name == 'platform') {
				$scope.types = [{
					name: 'openstack'
				}, {
					name: 'amazonec2'
				}, {
					name: 'google'
				}];
				$scope.platforms = [];
				$scope.$watch('currentPage', function() {
					$scope.getPlatforms();
				});
				$scope.recordPerPage = CommonService.recordNumPerPage();
			}
			if ($state.current.name == 'registry') {
				$scope.registries = [];
				$scope.$watch('currentPage', function() {
					$scope.getRegistries();
				});
				$scope.recordPerPage = CommonService.recordNumPerPage();
			}
		})();
		$scope.changeTab = function(target) {
			$location.path('/configuration/' + target);
		}
		$scope.totalrecords = 0;
		$scope.totalPage = 1;
		$scope.currentPage = 1;
		$scope.getKeyPairs = function() {
			var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
			var limit = $scope.recordPerPage;
			KeyPairService.getKeyPairs(skip, limit).then(function(data) {
					$scope.totalrecords = data.count;
					$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);

					$scope.keypairs = data.data;
				},
				function(error) {
					ResponseService.errorResponse(error, "config.key.listFailed");
				})
		};
		$scope.createKeyPair = function() {
			$uibModal.open({
					templateUrl: 'templates/configuration/key/create.html',
					controller: 'CreateKeyPairController',
					backdrop: 'static'
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						$scope.doCreate(response.data);
					}
				});
		};
		$scope.doCreate = function(keypair) {
			KeyPairService.createKeyPair(keypair).then(function(data) {
					$scope.downloadPrivateKey(data.data);
				},
				function(error) {
					ResponseService.errorResponse(error, "config.key.createFailed");
				})
		};
		$scope.downloadPrivateKey = function(data) {
			$uibModal.open({
				templateUrl: 'templates/configuration/key/download.html',
				controller: 'DownloadKeyPairController',
				backdrop: 'static'
			});
			window.location.assign("/keypair/download/" + data);
			$scope.getKeyPairs();
		}
		$scope.doDownload = function(userid) {
			KeyPairService.downloadKeyPair(userid).then(function(data) {
					$scope.getKeyPairs();
				},
				function(error) {
					ResponseService.errorResponse(error, "config.key.downloadFailed");
				})
		};
		$scope.uploadKeyPair = function() {
			$uibModal.open({
					templateUrl: 'templates/configuration/key/upload.html',
					controller: 'UploadKeyPairController',
					backdrop: 'static'
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						$scope.doUpload(response.data);
					}
				});
		};
		$scope.doUpload = function(keypair) {
			KeyPairService.uploadKeyPair(keypair).then(function(data) {
					$scope.getKeyPairs();
				},
				function(error) {
					ResponseService.errorResponse(error, "config.key.uploadFailed");
				})
		};
		$scope.confirmDeleteKey = function(key) {
			$scope.$translate(['common.deleteConfirm', 'config.key.deleteMessage', 'common.delete']).then(function(translations) {
				$scope.confirm = {
					"title": translations['common.deleteConfirm'],
					"message": translations['config.key.deleteMessage'],
					"button": {
						"text": translations['common.delete'],
						"action": function() {
							$scope.deleteKeyPair(key);
						}
					}

				};
				CommonService.deleteConfirm($scope);
			});
		};
		$scope.deleteKeyPair = function(key) {
			KeyPairService.deleteKeyPair(key).then(function(data) {
					$scope.getKeyPairs();
				},
				function(error) {
					ResponseService.errorResponse(error, "config.key.deleteFailed");
				})
		};
		$scope.$watch('selectedUser', function(nv, ov) {
			if (nv == ov) {
				return
			}
			$scope.getPlatforms();
		});
		$scope.getPlatforms = function() {
			var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
			var limit = $scope.recordPerPage;
			PlatformService.getProvider(limit, skip).then(function(data) {
					$scope.totalrecords = data.count;
					$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
					$scope.platforms = data.data;
				},
				function(error) {
					ResponseService.errorResponse(error);
				})
		};
		$scope.createPlatform = function() {
			$uibModal.open({
					templateUrl: 'templates/configuration/platform/create.html',
					controller: 'CreatePlatformController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								types: $scope.types
							};
						}
					}
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						switch (response.data.type) {
							case "openstack":
								response.data.openstackInfo['openstack-ssh-user'] = response.data.sshUser;
								break;
							case "amazonec2":
								response.data.awsEc2Info['amazonec2-ssh-user'] = response.data.sshUser;
								break;
							case "google":
								response.data.googleInfo["google-username"] = response.data.sshUser;
								response.data.googleInfo['google-application-credentials'] = Base64.encode(response.data.googleInfo['google-application-credentials']);

								// if internal ip is true, feedback true value to backend.
								// otherwise, return false
								if ( ! response.data.googleInfo['google-use-internal-ip'] || response.data.googleInfo['google-use-internal-ip'] == 'false') 
									response.data.googleInfo['google-use-internal-ip'] = "false";
								else
									response.data.googleInfo['google-use-internal-ip'] = "true";
								
								// if google tag is null, delete it
								if ( ! response.data.googleInfo["google-tags"] || response.data.googleInfo["google-tags"].length <= 0)
									delete response.data.googleInfo["google-tags"];
								break;
						}

						PlatformService.createProvider(response.data).then(function(data) {
								$scope.getPlatforms();
							},
							function(error) {
								ResponseService.errorResponse(error);
							});
					}
				});
		};
		$scope.editPlatform = function(platform) {
			$uibModal.open({
					templateUrl: 'templates/configuration/platform/edit.html',
					controller: 'EditPlatformController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								platform: platform,
								types: $scope.types
							};
						}
					}
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						switch (response.data.type) {
							case "openstack":
								response.data.openstackInfo['openstack-ssh-user'] = response.data.sshUser;
								break;
							case "amazonec2":
								response.data.awsEc2Info['amazonec2-ssh-user'] = response.data.sshUser;
								break;
							case "google":
								response.data.googleInfo["google-username"] = response.data.sshUser;
								response.data.googleInfo['google-application-credentials'] = Base64.encode(response.data.googleInfo['google-application-credentials']);

								// if internal ip is true, feedback true value to backend.
								// otherwise, return false
								if ( ! response.data.googleInfo['google-use-internal-ip'] || response.data.googleInfo['google-use-internal-ip'] == 'false') 
									response.data.googleInfo['google-use-internal-ip'] = "false";
								else
									response.data.googleInfo['google-use-internal-ip'] = "true";
								
								// if google tag is null, delete it
								if ( ! response.data.googleInfo["google-tags"] || response.data.googleInfo["google-tags"].length <= 0)
									delete response.data.googleInfo["google-tags"];
								break;
						}

						PlatformService.editProvider(response.data).then(function(data) {
								$scope.getPlatforms();
							},
							function(error) {
								ResponseService.errorResponse(error);
							});
					}
				});
		};
		$scope.deletePlatform = function(platform) {
			$scope.$translate(['common.deleteConfirm', 'config.platform.delete', 'common.delete']).then(function(translations) {
				$scope.confirm = {
					"title": translations['common.deleteConfirm'],
					"message": translations['config.platform.delete'],
					"button": {
						"text": translations['common.delete'],
						"action": function() {
							PlatformService.deleteProvider(platform).then(function(data) {
									$scope.getPlatforms();
								},
								function(error) {
									ResponseService.errorResponse(error);
								});
						}
					}
				};
				CommonService.deleteConfirm($scope);
			});
		};
		$scope.getSMTPServers = function() {
			var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
			var limit = $scope.recordPerPage;
			SMTPService.getSMTPServers(skip, limit).then(function(data) {
					$scope.totalrecords = data.count;
					$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
					$scope.smtpservers = data.data;
				},
				function(error) {
					ResponseService.errorResponse(error, "config.smtp.listFailed");
				})
		};
		$scope.addSMTPServer = function() {
			$uibModal.open({
					templateUrl: 'templates/configuration/smtp/add.html',
					controller: 'AddSMTPController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								smtp: {
									"name": "",
									"passwd": "",
									"address": ""
								},
								type: "add"
							}
						}
					}
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						$scope.doAdd(response.data);
					}
				});
		};
		$scope.doAdd = function(smtp) {
			SMTPService.addSMTPServer(smtp).then(function(data) {
					$scope.getSMTPServers();
				},
				function(error) {
					ResponseService.errorResponse(error, "config.smtp.addFailed");
				})
		};
		$scope.editSMTPServer = function(smtp) {
      var oldPasswd = smtp.passwd;
			$uibModal.open({
					templateUrl: 'templates/configuration/smtp/add.html',
					controller: 'AddSMTPController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								smtp: {
									"_id": smtp._id,
									"name": smtp.name,
									"passwd": smtp.passwd,
									"address": smtp.address
								},
								type: "edit"
							};
						}
					}
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
            // Set passwd to empty(backend will ignore) if user did not change it
            if (response.data.passwd === oldPasswd) {
              response.data.passwd = "";
            }
						$scope.doEdit(response.data);
					}
				});
		};
		$scope.doEdit = function(smtp) {
			SMTPService.editSMTPServer(smtp).then(function(data) {
					$scope.getSMTPServers();
				},
				function(error) {
					ResponseService.errorResponse(error, "config.smtp.updateFailed");
				})
		};
		$scope.confirmDeleteSMTPServer = function(smtp) {
			$scope.$translate(['common.deleteConfirm', 'config.smtp.deleteMessage', 'common.delete']).then(function(translations) {
				$scope.confirm = {
					"title": translations['common.deleteConfirm'],
					"message": translations['config.smtp.deleteMessage'],
					"button": {
						"text": translations['common.delete'],
						"action": function() {
							$scope.deleteSMTPServer(smtp);
						}
					}

				};
				CommonService.deleteConfirm($scope);
			});
		};
		$scope.deleteSMTPServer = function(smtp) {
			SMTPService.deleteSMTPServer(smtp).then(function(data) {
					$scope.getSMTPServers();
				},
				function(error) {
					ResponseService.errorResponse(error, "config.smtp.deleteFailed");
				})
		};
		$scope.getRegistries = function() {
			var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
			var limit = $scope.recordPerPage;
			DockerregistriesService.getDockerregistries(skip, limit).then(function(data) {
					$scope.totalrecords = data.count;
					$scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
					$scope.registries = data.data;
				},
				function(error) {
					ResponseService.errorResponse(error);
				})
		};
		$scope.createRegistry = function() {
			$uibModal.open({
					templateUrl: 'templates/configuration/registry/add.html',
					controller: 'createRegistryController',
					backdrop: 'static'
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						DockerregistriesService.createDockerregistry(response.data).then(function(data) {
								$scope.getRegistries();
							},
							function(error) {
								ResponseService.errorResponse(error);
							})
					}
				});
		};
		$scope.detailRegistry = function(item) {
			$uibModal.open({
					templateUrl: 'templates/configuration/registry/detail.html',
					controller: 'detailRegistryController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								registry: item
							};
						}
					}
			});
		};
		$scope.deleteRegistry = function(registry) {
			DockerregistriesService.validateName(registry.name,'used').then(function(data) {
				$scope.$translate(['common.deleteConfirm', 'config.registry.delete', 'common.delete']).then(function(translations) {
					$scope.confirm = {
						"title": translations['common.deleteConfirm'],
						"message": translations['config.registry.delete'],
						"button": {
							"text": translations['common.delete'],
							"action": function() {
								DockerregistriesService.deleteDockerregistry(registry._id).then(function(data) {
										$scope.getRegistries();
									},
									function(error) {
										ResponseService.errorResponse(error);
									});
							}
						}
					};
					CommonService.deleteConfirm($scope);
				});
			},
			function(error) {
				ResponseService.errorResponse(error);
			});
		};
		$scope.editRegistry = function(registry) {
			$uibModal.open({
					templateUrl: 'templates/configuration/registry/edit.html',
					controller: 'editRegistryController',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								registry: registry
							};
						}
					}
				})
				.result
				.then(function(response) {
					if (response.operation === 'execute') {
						console.log(response.data)
					}
				});
		}
	}]);
	app.controllerProvider.register('CreateKeyPairController', ['$scope', '$uibModalInstance',
		function($scope, $uibModalInstance) {
			$scope.keypair = {
				"name": ""
			};

			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.keypair
				});
			};
		}
	]);
	app.controllerProvider.register('DownloadKeyPairController', ['$scope', '$uibModalInstance',
		function($scope, $uibModalInstance) {
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res
				});
			};
		}
	]);
	app.controllerProvider.register('UploadKeyPairController', ['$scope', '$uibModalInstance',
		function($scope, $uibModalInstance) {
			$scope.keypair = {
				"name": "",
				"pubkey_value": ""
			};

			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.keypair
				});
			};
		}
	]);
	app.controllerProvider.register('CreatePlatformController', ['$scope', '$uibModalInstance', 'model', 'PlatformService',
		function($scope, $uibModalInstance, model, PlatformService) {
			$scope.validate = true;
			$scope.types = model.types;
			$scope.platform = {
				"type": $scope.types[0].name,
				"sshUser": "",
				"openstackInfo": {
					"openstack-auth-url": "",
					"openstack-username": "",
					"openstack-password": "",
					"openstack-tenant-name": "",
					"openstack-flavor-name": "",
					"openstack-image-name": "",
					"openstack-sec-groups": "",
					"openstack-floatingip-pool": "",
					"openstack-nova-network": "",
					"openstack-ssh-user":""
				},
				"awsEc2Info": {
					"amazonec2-access-key": "",
					"amazonec2-secret-key": "",
					"amazonec2-ami": "",
					"amazonec2-instance-type": "",
					"amazonec2-root-size": "",
					"amazonec2-region": "",
					"amazonec2-vpc-id": "",
					"amazonec2-ssh-user":""
				},
				"googleInfo":{
			       "google-project":"",
			       "google-zone":"",
			       "google-machine-type":"",
			       "google-machine-image":"",
			       "google-network":"",
			       "google-username":"",
			       "google-disk-size":"",
			       "google-disk-type":"",
			       "google-use-internal-ip":"true",
			       "google-tags":"",
			       "google-application-credentials":""
			    }
			};
			$scope.validateName = function() {
				var platformname = $scope.platform.name;
				if (!angular.isUndefined(platformname) && !_.isEmpty(platformname)) {
					PlatformService.validateName(platformname).then(function(data) {
							$scope.validate = true;
						},
						function(errorMessage) {
							$scope.validate = false;
						});
				} else {
					$scope.validate = true;
				}

			};
			$scope.$watch('platform.type', function() {
				if ($scope.platform.type == "openstack") {
					for (var i in $scope.platform.openstackInfo) {
						$scope.platform.openstackInfo[i] = "";
					}
				} else {
					for (var i in $scope.platform.awsEc2Info) {
						$scope.platform.awsEc2Info[i] = "";
					}
				}

			});
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.platform
				});
			};
		}
	]);
	app.controllerProvider.register('EditPlatformController', ['$scope', '$uibModalInstance', 'model',
		function($scope, $uibModalInstance, model) {
			$scope.types = model.types;
			$scope.platform = {};

			angular.copy(model.platform, $scope.platform);
			if ($scope.platform.type == 'google') {
				$scope.platform.googleInfo['google-application-credentials'] = Base64.decode($scope.platform.googleInfo['google-application-credentials']);
				if ($scope.platform.googleInfo && $scope.platform.googleInfo['google-use-internal-ip'] == 'true')
					$scope.platform.googleInfo['google-use-internal-ip'] = 'true';
				else
					$scope.platform.googleInfo['google-use-internal-ip'] = 'false';	
			}

			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.platform
				});
			};
		}
	]);
	app.controllerProvider.register('DeletePlatformController', ['$scope', '$uibModalInstance', 'model',
		function($scope, $uibModalInstance, model) {
			$scope.platform = model.platform;
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.platform
				});
			};
		}
	]);
	app.controllerProvider.register('AddSMTPController', ['$scope', '$uibModalInstance', 'model',
		function($scope, $uibModalInstance, model) {
			$scope.smtp = model.smtp;
			$scope.type = model.type;

			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.smtp
				});
			};
		}
	]);
	app.controllerProvider.register('createRegistryController', ['$scope', '$uibModalInstance','DockerregistriesService',
		function($scope, $uibModalInstance,DockerregistriesService) {
			$scope.validate = true;
			$scope.validateRegistry = function() {
				var name = $scope.registry.name;
				if (!angular.isUndefined(name) && !_.isEmpty(name)) {
					DockerregistriesService.validateName(name,'name').then(function(data) {
							$scope.validate = true;
						},
						function(errorMessage) {
							$scope.validate = false;
						});
				} else {
					$scope.validate =true;
				}
			};
			$scope.$watch('registry',function(nv,ov){
				if(nv==ov)return;
				if(nv.secure==false){
					$scope.registry.ca_text='';
				}
				if($scope.registry.password && !$scope.registry.username){
					$scope.usernameInfo=true;
				}else{
					$scope.usernameInfo=false;
				}
			},true);
			$scope.usernameInfo=false;
			$scope.registry = {
				name: '',
				registry: '',
				secure: true,
				ca_text:'',
				username:'',
				password:''
			}
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.registry
				});
			};
		}
	]);
	app.controllerProvider.register('detailRegistryController', ['$scope', '$uibModalInstance', 'model',
		function($scope, $uibModalInstance, model) {
			$scope.registry = model.registry;
			$scope.close = function(res) {
				$uibModalInstance.close({
					"operation": res,
					"data": $scope.registry
				});
			};
		}
	]);
	app.controllerProvider.register('editRegistryController', ['$scope', '$uibModalInstance', 'model',
		function($scope, $uibModalInstance, model) {
			$scope.registry = model.registry;
			$scope.close = function(res) {
				$uibModalInstance.close();
			};
		}
	]);
});