define(['app', 'services/node/main','services/network/main','services/service/main', 'services/serviceGraph/main', 'services/serviceGraph/dataParser', 'services/serviceGraph/groupParser', 'services/service/component', 'services/service/container', 'directives/status'], function(app) {
    'use strict';
    app.controllerProvider.register('ServiceDetailController', ['$scope','$stateParams','$timeout','$q','$uibModal', '$state', '$location','$localStorage', 'ServiceService', 'CommonService', 'ResponseService', 'ComponentService','ContainerService', 'ServiceGraphService', 'treeUtilService', 'componentTreeUtilService', 'NodeService',function($scope,$stateParams,$timeout,$q, $uibModal, $state, $location, $localStorage, ServiceService, CommonService, ResponseService, ComponentService,ContainerService, ServiceGraphService, treeUtilService, componentTreeUtilService,NodeService) {

        $scope.$storage = $localStorage;
        $scope.$watch('$storage.cluster', function(newValue, oldValue) {
            if (!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id) {
                $state.go("service");
            }

        }, true);

        $scope.containerObj = {};
        $scope.selectedService = {
        	'status':$state.params.serviceStatus
        };
        $scope.jsonString = "";
        $scope.showDetail = function() {
            if (CommonService.endPointAvailable()) {
                ServiceService.getDetail(CommonService.getEndPoint(), $state.params.serviceName).then(function(data) {
                        $scope.dataReady = true;
                        $scope.selectedService = data.data;
                        $scope.getContainerName(data.data.components);
                        $state.transitionTo('service.detail', {serviceStatus: data.data.status,serviceName:$stateParams.serviceName}, { location: true, inherit: true, relative: $state.$current, notify: false })
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.componentListFailed");
                    })
            }
        };
        $scope.getContainerName = function(components){
             _.each(components, function(component) {
                    var tasks = component.app.tasks;
                    _.each(tasks,function(task){
                        var taskid = task.id;
                        var slaveid = task.slaveId;
                        ContainerService.getName(CommonService.getEndPoint(), slaveid, taskid).then(function(data) {
                            $scope.containerObj[taskid] = CommonService.generateContainerName(slaveid,data);
                        }, function(error) {
                            ResponseService.errorResponse(error, "service.componentListFailed");
                        })
                    })

             })
        };
        $scope.createComponent = function() {
            $uibModal.open({
                    templateUrl: 'templates/service/createComponent.html',
                    controller: 'CreateComponentController',
                    backdrop: 'static',
                    size: 'lg',
                    resolve: {
                        model: function() {
                            return {
                                components: $scope.selectedService.components || [],
                                saveComponent:$scope.saveComponent
                            };
                        }
                    }
               })
        };
        $scope.saveComponent = function(component, type) {
          var data = _.cloneDeep(component);
          // MESOS doesn't allow "network" parameter
          if (data.app.container.type === "MESOS") {
            delete data.app.container.docker.network;
          }
        	var deferred = $q.defer();
            if (CommonService.endPointAvailable()) {
                ComponentService.save(CommonService.getEndPoint(), data, type).then(function(data) {
                		deferred.resolve()
                        $timeout(function(){$scope.showDetail();},200)
                    },
                    function(error) {
                    	if(error.code=="E70102"||error.code=="E70103"){
                    		deferred.reject();
                    	}else{
                    		deferred.resolve();
                    	}
                        ResponseService.errorResponse(error, "service.componentsaveFailed");
                    })
            }
            return deferred.promise
        };
        $scope.showConsole = function(item){
        	window.open('/webconsole?cid='+item+'&clientAddr='+CommonService.getEndPoint());
        }
        $scope.prepareScale = function(component) {
            $uibModal.open({
                    templateUrl: 'templates/service/scaleContainer.html',
                    controller: 'ScaleContainerController',
                    backdrop: 'static',
                    // size: 'lg',
                    resolve: {
                        model: function() {
                            return {
                                component: component
                            };
                        }
                    }
                })
                .result
                .then(function(response) {
                    if (response.operation === 'execute') {
                        $scope.scaleContainer(response.data);
                    }
                });
        };
        $scope.scaleContainer = function(info) {
            if (CommonService.endPointAvailable()) {
                ComponentService.scale(CommonService.getEndPoint(), info).then(function(data) {
                        $timeout(function(){$scope.showDetail();},200)
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.componentscaleFailed");
                    })
            }
        };
        $scope.prepareEdit = function(component) {
        	var components=_.filter($scope.selectedService.components,function(item){
        		return item.app.id!=component.app.id
        	});
        	var groups=_.filter($scope.selectedService.group.apps,function(item){
        		return item.id!=component.app.id
        	})
            $uibModal.open({
                    templateUrl: 'templates/service/editComponent.html',
                    controller: 'EditComponentController',
                    backdrop: 'static',
                    size: 'lg',
                    resolve: {
                        model: function() {
                            return {
                                component: component,
                                components: components || [],
                                groups : groups || [],
                                saveComponent : $scope.saveComponent
                            };
                        }
                    }
               })
        };
        $scope.startOrStopComponent = function(component, type){
            if (CommonService.endPointAvailable()) {
                ComponentService.startOrStop(CommonService.getEndPoint(), component.app.id, type).then(function(data) {
                        $timeout(function(){$scope.showDetail();},200)
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.component"+type+"Failed");
                    })
            }
        };
        $scope.confirmDelete = function(component) {
            $scope.$translate(['common.deleteConfirm', 'service.componentDeleteMessage', 'common.delete']).then(function(translations) {
                $scope.confirm = {
                    "title": translations['common.deleteConfirm'],
                    "message": translations['service.componentDeleteMessage'],
                    "button": {
                        "text": translations['common.delete'],
                        "action": function() {
                            $scope.deleteComponent(component);
                        }
                    }

                };
                CommonService.deleteConfirm($scope);
            });
        };
        $scope.deleteComponent = function(component) {
            ComponentService.delete(CommonService.getEndPoint(),component.app.id).then(function(data) {
                    $timeout(function(){$scope.showDetail();},200)
                },
                function(error) {
                    ResponseService.errorResponse(error, "service.componentdeleteFailed");
                })
        };
        $scope.confirmOperate = function(containerId,type) {
            $scope.$translate(['common.operateConfirm', 'service.container'+type+'Message', 'common.'+type]).then(function(translations) {
                $scope.confirm = {
                    "title": translations['common.operateConfirm'],
                    "message": translations['service.container'+type+'Message'],
                    "button": {
                        "text": translations['common.'+type],
                        "action": function() {
                            $scope.operate(containerId,type);
                        }
                    }

                };
                CommonService.deleteConfirm($scope);
            });
        };
        $scope.operate = function(containerId,type){
            if (CommonService.endPointAvailable()) {
                ContainerService.operate(CommonService.getEndPoint(), containerId,type).then(function(data) {
                        $timeout(function(){$scope.showDetail();},200)
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.containeroperateFailed");
                    })
            }
        };


        $scope.zoomingRate = 0.85;
        $scope.showJSON = function() {
            $scope.selectedModel = $scope.selectedService.group;
            $scope.jsonString = JSON.stringify($scope.selectedModel, undefined, 2);
        };
        $scope.showGraph = function() {
            $scope.selectedModel = $scope.selectedService.group;
            var isNested = false;
            if(!_.isUndefined($scope.selectedModel.groups) && $scope.selectedModel.groups.length>0){
                isNested = true;
            }

            treeUtilService.allocateImageToApp($scope.selectedModel);
            $scope.modelDesignerHeight = 560;

            //parse service json
            $scope.nodes = [];
            $scope.relations = [];
            $scope.extra_relations = [];

            //for data with groups
            if(isNested){
                $scope.levelHeights = [];
            }

            //get root path
            var rootPathID = $scope.selectedModel.id;
            //parse selectedModel json
            if(!isNested){
                $scope.relations = componentTreeUtilService.parseGroup(rootPathID, $scope.selectedModel, $scope.nodes, $scope.relations, $scope.extra_relations, $scope.zoomingRate);
            }else{
                //for data with groups
                $scope.relations = treeUtilService.parseService(rootPathID, $scope.selectedModel, $scope.nodes, $scope.relations, $scope.extra_relations, $scope.levelHeights, $scope.zoomingRate, "model");
            }

            var treejson = _.find($scope.nodes, function(node) {
                return node.id == rootPathID;
            })


            setTimeout(function() {
                if(!isNested){
                    ServiceGraphService.drawComponentGraph("serviceGraphDiv", treejson, $scope);
                }else{
                    //for data with groups
                    ServiceGraphService.drawGraph("serviceGraphDiv", treejson, $scope);
                }
            }, 500)
        };

        // component relations
        $scope.checkIfComponentsHaveDependencies = function(group) {
            var result = false;
            _.each(group.apps, function(app) {
                if (!_.isUndefined(app.dependencies)) {
                    result = true;
                }
            })
            return result;
        }

        $scope.showComponentDependencies = function(event) {
            var groupid = $(event.currentTarget).parent().find(".group-label").data("groupid");
            var group = _.find($scope.nodes, function(node) {
                return node.id == groupid;
            }).data;

            //parse service json
            $scope.c_nodes = [];
            $scope.c_relations = [];
            $scope.c_extra_relations = [];

            //get root path
            var rootPathID = group.pathid;
            //parse selectedModel json
            $scope.c_relations = componentTreeUtilService.parseGroup(rootPathID, group, $scope.c_nodes, $scope.c_relations, $scope.c_extra_relations, $scope.zoomingRate);

            var treejson = _.find($scope.c_nodes, function(node) {
                return node.id == rootPathID;
            })

            $uibModal.open({
                templateUrl: 'templates/service/componentRelationGraph.html',
                controller: 'ComponentRelationController',
                backdrop: 'static',
                size: 'lg',
                resolve: {
                    model: function() {
                        return {
                            data: treejson,
                            reuselines: $scope.c_extra_relations
                        };
                    }
                }
            });
        }

        //zoom
        $scope.zoomin = function() {
            $scope.zoomingRate = $scope.zoomingRate  / 0.9;
            $scope.drawServiceGroup();
        };
        $scope.zoomout = function() {
            $scope.zoomingRate = $scope.zoomingRate * 0.9;
            $scope.drawServiceGroup();
        };

        $scope.getZoomingValue = function(target, property) {
            return treeUtilService.getZoomingValue(target, property, $scope.zoomingRate);
        };

        $scope.getZoomingCssString = function(target) {
            return treeUtilService.getZoomingCssString(target, $scope.zoomingRate);
        };
        $scope.editJSON = function(){
              $uibModal.open({
                    templateUrl: 'templates/service/editJSON.html',
                    controller: 'EditJSONController',
                    backdrop: 'static',
                    size:"lg",
                    resolve: {
                        model: function() {
                            return {
                                service: $scope.selectedService
                            };
                        }
                    }
                })
                .result
                .then(function(response) {
                    if (response.operation === 'execute') {
                        $scope.saveService(response.data);
                    }
                });
        };
        $scope.saveService = function(service) {
            if (CommonService.endPointAvailable()) {
                ServiceService.update(CommonService.getEndPoint(),service).then(function(data) {
                        $timeout(function(){$scope.showDetail();},200)
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.updateFailed");
                    })
            }
        };

        $scope.idToSimple = function(id){
            return treeUtilService.idToSimple(id);
        };
        $scope.showContainerInfo = function(containerId,slaveId){
            $uibModal.open({
                    templateUrl: 'templates/service/container.html',
                    controller: 'ContainerController',
                    backdrop: 'static',
                    size:"lg",
                    resolve: {
                        model: function() {
                            return {
                                containerId: containerId,
                                slaveId:slaveId
                            };
                        }
                    }
                });

        };
        $scope.goToMonitoring = function(containerName){
           $localStorage.selectedServiceToMonitor = $scope.selectedService.name
           $localStorage.selectedContainerToMonitor = containerName;
           $location.path("/monitor");
        };

        $scope.dataReady = false;
        $scope.showDetail();

    }]);

    app.controllerProvider.register('ComponentRelationController', ['$scope', '$uibModalInstance', 'model', 'ServiceGraphService', 'treeUtilService',
        function($scope, $uibModalInstance, model, ServiceGraphService, treeUtilService) {
            $scope.extra_relations = model.reuselines;

            $scope.zoomingRate = 1;
            //zoom
            $scope.zoomin = function() {
                $scope.zoomingRate = $scope.zoomingRate  / 0.9;
                $scope.drawComponentGraph();
            };
            $scope.zoomout = function() {
                $scope.zoomingRate = $scope.zoomingRate * 0.9;
                $scope.drawComponentGraph();
            };

            $scope.getZoomingValue = function(target, property) {
                return treeUtilService.getZoomingValue(target, property, $scope.zoomingRate);
            };

            $scope.getZoomingCssString = function(target) {
                return treeUtilService.getZoomingCssString(target, $scope.zoomingRate);
            };

            $scope.idToSimple = function(id){
                return treeUtilService.idToSimple(id);
            }

            $scope.drawComponentGraph = function() {
                ServiceGraphService.drawComponentGraph("componentGraphDiv", model.data, $scope);
            };

            $scope.close = function() {
                $uibModalInstance.close();
            };

            setTimeout(function() {
                $scope.drawComponentGraph();
            }, 500)
        }
    ]);

    app.controllerProvider.register('CreateComponentController', ['$scope', '$uibModalInstance', '$state','$localStorage', '$q','ComponentService','NetworkService','CommonService', 'ServiceService','model',
        function($scope, $uibModalInstance, $state, $localStorage,$q,ComponentService,NetworkService,CommonService,ServiceService,model) {
            var componentJSON_1 = ComponentService.componentJSON().JSONTemplate;
            var portMapping_1 = ComponentService.componentJSON().portMapping;
            var volume_1 = ComponentService.componentJSON().volume;
			var ExternalAccess_1=angular.copy(portMapping_1);
			ExternalAccess_1.hostPort=0;
			$scope.ExternalAccess={
				checked:false,
				array:[angular.copy(ExternalAccess_1)]
			}
			$scope.$storage = $localStorage;
            $scope.componentJSON = angular.copy(componentJSON_1);
            // init uris and acceptedResourceRoles
            $scope.componentJSON.marathon_app.uris = $scope.componentJSON.marathon_app.uris.join();
            $scope.componentJSON.marathon_app.acceptedResourceRoles = $scope.componentJSON.marathon_app.acceptedResourceRoles.join();

            $scope.component = { "appset_name": $state.params.serviceName, "app": $scope.componentJSON.marathon_app };

            $scope.portArray = [];
            $scope.volumeArray = [angular.copy(volume_1)];

            $scope.paramsArray = [{ "key": "", "value": "" }];

            $scope.labelArray = [{ "key": "", "value": "" }];
            $scope.constraintArray = [{ "key": "", "constraint": "UNIQUE", "value": "" }];
            $scope.relationshipArray = [];
            $scope.argArray = [{ "value": "" }];
			$scope.UserDefinedNetwork=[];
			$scope.UDNetwork={
				'NetworkName':'',
				'NetworkDomain':''
			}

            // select alarm data
            $scope.alarmObj = { "upperLimit": "", "lowerLimit": "", "cpuUpperLimit": "","cpuLowerLimit":"", "memoryUpperLimit": "","memoryLowerLimit":"" };
            ServiceService.getapps(CommonService.getEndPoint(),$scope.component.appset_name).then(function(data){
                $scope.ScaledAppIds=data.data;
                $scope.alarmObj.SCALED_APP_ID=null;
                if($scope.ScaledAppIds && $scope.ScaledAppIds.indexOf($scope.componentJSON.marathon_app.id)==-1){
                    $scope.alarmObj.SCALED_APP_ID=$scope.ScaledAppIds[0];
                }
            });

        	$scope.resetUDNetwork=function(){
        		$scope.UDNetwork={
					'NetworkName':(function(){
						if($scope.UserDefinedNetwork[0]&&$scope.UserDefinedNetwork[0].network){
							return $scope.UserDefinedNetwork[0].network.name||""
						}else return ""
					})(),
					'NetworkDomain':''
				}
        	};
        	$scope.changeDependency = function(){
        		_.each($scope.componentList,function(item){
        			if(item.app.id==$scope.dependency.selectedComponent){
        				$scope.dependency.servicePort=item.app.container.docker.portMappings||[];
        				$scope.dependency.show= _.some(item.app.labels,function(item){
            				return item=="linkermgmt"
            			})
        				$scope.dependency.domain = _.filter(item.app.container.docker.parameters,function(item){
		            		if(item.key=="net-alias"){
		            			return item.value
		            		}
		            	})||[];
        			}
        		})
        	}
        	$scope.resetExternalAccess=function(){
        		$scope.ExternalAccess.array = [angular.copy(ExternalAccess_1)];
        	}
        	NetworkService.getNetwork(CommonService.getEndPoint(), 0, 0, $scope.$storage.cluster._id)
        		.then(function(data){
        			_.each(data.data,function(item){
        				if(item.network.driver=="overlay")
        				$scope.UserDefinedNetwork.push(item);
        			})
        		});

			$scope.setEnv = function() {
				if(model.components && model.components.length > 0 && !$scope.componentJSON.marathon_app.container.docker.network) {
					var temp = _.filter(model.components, function(item) {
                        console.log(item);
						if($scope.dependency.selectedComponent == item.app.id) {
							return item
						}
					})[0];
					if(temp.app.container.docker.network=="BRIDGE") {
						return _.chain(temp.app.container.docker.parameters).map(function(item){
							if(item.key!='net'||(item.key == 'net' && item.value == $scope.UDNetwork.NetworkName)) {
								return true
							}else{
								return false
							}
						}).every()._wrapped;
					}
				}
				return false
			}
			$scope.componentList=model.components;
            $scope.envArray = {
            	'relationship':{},
            	'advanced':[{ "key": "", "value": "" }]
            };
            $scope.clearUDNetArray=function(){
            	$scope.componentList.forEach(function(item){
	            	$scope.envArray.relationship[item.app.id]=new Array();
	           	})
            };
            $scope.clearUDNetArray();

            $scope.dependency = {
            	"selectedComponent": (function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app){
            			return $scope.componentList[0].app.id||""
            		}else return ""
            	})(),
            	"show":(function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app&&$scope.componentList[0].app.labels){
            			return _.some($scope.componentList[0].app.labels,function(item){
            				return item=="linkermgmt"
            			})
            		}else return false
            	})(),
            	"servicePort":(function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app&&$scope.componentList[0].app.container&&$scope.componentList[0].app.container.docker){
            			return $scope.componentList[0].app.container.docker.portMappings||[]
            		}else return []
            	})(),
            	"domain":(function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app&&$scope.componentList[0].app.container&&$scope.componentList[0].app.container.docker){
            			return _.filter($scope.componentList[0].app.container.docker.parameters,function(item){
            						if(item.key=="net-alias"){
            							return item.value
            						}
            					})||[]
            		}else return []
            	})()
            };
            $scope.advance = { "isAlarm": false };
            $scope.alarmObj = { "upperLimit": "", "lowerLimit": "", "cpuUpperLimit": "","cpuLowerLimit":"", "memoryUpperLimit": "","memoryLowerLimit":"" };

            $scope.stepObj = { "previous": "", "current": "basic", "next": "network" };
            $scope.goNext = function(current) {
                $scope.stepObj = ComponentService.goNext(current);
            };
            $scope.goPrevious = function(current) {
                $scope.stepObj = ComponentService.goPrevious(current);
            };
            $scope.setCurrent = function(current) {
                $scope.stepObj = ComponentService.setCurrent(current);
            };
            $scope.addParameter = function(type) {
                switch (type) {
                    case "port":
                        $scope.portArray.push(angular.copy(portMapping_1));
                        break;
                    case "envRelation":
                        $scope.envArray.relationship[arguments[1]].push({ "key": "", "value": "" });
                        break;
                    case "envAdvanced":
                        $scope.envArray.advanced.push({ "key": "", "value": "" });
                        break;
                    case "parameters":
                        $scope.paramsArray.push({ "key": "", "value": "" });
                        break;
                    case "volume":
                        $scope.volumeArray.push(angular.copy(volume_1));
                        break;
                    case "externalaccess":
                    	$scope.ExternalAccess.array.push(angular.copy(ExternalAccess_1));
                    	break;
                    case "constraint":
                        $scope.constraintArray.push({ "key": "", "constraint": "UNIQUE", "value": "" });
                        break;
                    case "relationship":
                    	if($scope.relationshipArray.indexOf($scope.dependency.selectedComponent)==-1){
							 $scope.relationshipArray.push($scope.dependency.selectedComponent);
                    	}
                        break;
                    case "arg":
                        $scope.argArray.push({ "value": "" });
                        break;
                    case "label":
                        $scope.labelArray.push({ "key": "", "value": "" });
                        break;
                }
            };
            $scope.removeParameter = function(type, index) {
                switch (type) {
                    case "port":
                        $scope.portArray.splice(index, 1);
                        break;
                    case "envRelation":
                        $scope.envArray.relationship[arguments[2]].splice(index,1);
                        break;
                    case "envAdvanced":
                        $scope.envArray.advanced.splice(index,1);
                        break;
                    case "parameters":
                        $scope.paramsArray.splice(index, 1);
                        break;
                    case "volume":
                        $scope.volumeArray.splice(index, 1);
                        break;
                    case "externalaccess":
                    	$scope.ExternalAccess.array.splice(index,1);
                    	break;
                    case "constraint":
                        $scope.constraintArray.splice(index, 1);
                        break;
                    case "relationship":
                        $scope.relationshipArray.splice(index, 1);
                        break;
                    case "arg":
                        $scope.argArray.splice(index, 1);
                        break;
                    case "label":
                        $scope.labelArray.splice(index, 1);
                        break;
                }
            };

            $scope.addHealthCheck = function() {
                if ( !$scope.component.app.healthChecks )
                    $scope.component.app.healthChecks = [];
                $scope.component.app.healthChecks.push(angular.copy( ComponentService.componentJSON().HTTP ));
                // console.log( $scope.component.app.healthChecks );
            }

            $scope.removeHealCheck = function( index ) {
                if ( $scope.component.app.healthChecks && index > -1 && index < $scope.component.app.healthChecks.length )
                    $scope.component.app.healthChecks.splice( index, 1 );
            }

            $scope.changeHealthCheck = function( index, protocol ) {
                $scope.component.app.healthChecks[index] = angular.copy( ComponentService.componentJSON()[protocol] );
                // console.log($scope.component.app.healthChecks[index]);
            }

            $scope.changePortType = function( item ) {
                if ( item.portType == 'PORT_NUMBER' ) {
                    delete item.portIndex;
                    item.port = 0;
                } // if
                else {
                    delete item.port;
                    item.portIndex = 0;
                }
            }

            $scope.changeRuntime = function() {
                $scope.componentJSON.marathon_app.container.docker.network = "HOST";
                $scope.resetUDNetwork();
                $scope.clearUDNetArray();
                // Docker doesn't support gpu
                if ($scope.componentJSON.marathon_app.container.type == 'DOCKER') {
                    $scope.componentJSON.marathon_app.gpus = "";
                }
            }

            $scope.assembleData = function() {
                $scope.componentJSON.marathon_app.container.docker.portMappings = $scope.portArray;
                if($scope.ExternalAccess.checked){
                	$scope.componentJSON.marathon_app.container.docker.portMappings=$scope.componentJSON.marathon_app.container.docker.portMappings.concat($scope.ExternalAccess.array)
                }
                $scope.componentJSON.marathon_app.container.volumes = _.filter($scope.volumeArray,function(item){
                   return item.containerPath != "" && item.hostPath != "";
                });

                // add alarmObj data to advance alarm items.
                if ($scope.advance.isAlarm) {
                    var tempArray = [{ "key": "INSTANCE_MAX_NUM", "value": $scope.alarmObj.upperLimit },
                        { "key": "INSTANCE_MIN_NUM", "value": $scope.alarmObj.lowerLimit },
                        { "key": "CPU_USAGE_HIGH_THRESHOLD", "value": $scope.alarmObj.cpuUpperLimit },
                        { "key": "MEMORY_USAGE_HIGH_THRESHOLD", "value": $scope.alarmObj.memoryUpperLimit },
                        { "key": "CPU_USAGE_LOW_THRESHOLD", "value": $scope.alarmObj.cpuLowerLimit },
                        { "key": "MEMORY_USAGE_LOW_THRESHOLD", "value": $scope.alarmObj.memoryLowerLimit },
                        { "key": "SCALED_APP_ID", "value": $scope.alarmObj.SCALED_APP_ID},
                        { "key": "SCALE_NUMBER", "value": $scope.alarmObj.SCALE_NUMBER},
                        { "key": "ALERT_ENABLE", "value": "true" }
                    ];
                    $scope.envArray.advanced = $scope.envArray.advanced.concat(tempArray);
                }
                var tempEnvArray = _.filter($scope.envArray.advanced,function(item){return item.key != "";});
                for(var i in $scope.envArray.relationship){
                	tempEnvArray=tempEnvArray.concat( _.filter($scope.envArray.relationship[i],function(item){return item.key != "";}))
                }
                var envKeys = _.map(tempEnvArray, 'key');
                var envValues = _.map(tempEnvArray, 'value');
                $scope.componentJSON.marathon_app.env = ComponentService.handleSpecialItem(_.zipObject(envKeys, envValues));
                _.each($scope.paramsArray,function(item){
                	var tempKey = item.key;
                	if(tempKey != ""){
                    $scope.componentJSON.marathon_app.container.docker.parameters.push(item);
                    }
                });
                if(!$scope.componentJSON.marathon_app.container.docker.network){
                	$scope.componentJSON.marathon_app.container.docker.parameters.push(
                	{
                		"key":"net",
                		"value":$scope.UDNetwork.NetworkName,
                		"description":""
                	},{
                		"key":"net-alias",
                		"value":$scope.UDNetwork.NetworkDomain,
                		"description":""
                	});
                	$scope.componentJSON.marathon_app.container.docker.network="BRIDGE"
                }

                var tempLabelArray = _.filter($scope.labelArray,function(item){return item.key != "";});
                if($scope.ExternalAccess&&$scope.ExternalAccess.checked){
                	tempLabelArray.push({ "key": "HAPROXY_GROUP", "value": "linkermgmt" })
                }
                var labelKeys = _.map(tempLabelArray, 'key');
                var labelValues = _.map(tempLabelArray, 'value');
                $scope.componentJSON.marathon_app.labels = _.zipObject(labelKeys, labelValues);
                var finalConstraintArray = [];
                _.each($scope.constraintArray, function(constraint) {
                    var tempArray = [];
                    if (constraint.key != "") {
                        tempArray.push(constraint.key);
                        tempArray.push(constraint.constraint);
                        if(constraint.value != ""){
                             tempArray.push(constraint.value);
                         }
                        finalConstraintArray.push(tempArray);
                    }

                })
                $scope.componentJSON.marathon_app.cmd = ComponentService.handleSpecialItem($scope.componentJSON.marathon_app.cmd);
                $scope.componentJSON.marathon_app.constraints = finalConstraintArray;
                $scope.componentJSON.marathon_app.dependencies = $scope.relationshipArray;
                $scope.componentJSON.marathon_app.args = ComponentService.handleSpecialItem(_.without(_.map($scope.argArray, 'value'),""));
                $scope.componentJSON.marathon_app.uris = $scope.component.app.uris.length > 0 ? $scope.component.app.uris.split(',') : [];
                $scope.componentJSON.marathon_app.executor = $scope.component.app.executor;
                $scope.componentJSON.marathon_app.acceptedResourceRoles = $scope.component.app.acceptedResourceRoles.length > 0 ? $scope.component.app.acceptedResourceRoles.split(',') : [];
                $scope.componentJSON.marathon_app.healthChecks = $scope.component.app.healthChecks;
            };
            $scope.close = function(res) {
                if (res === "execute") {
                    $scope.assembleData();
                    model.saveComponent($scope.component, "create").then(function(){
                    	$uibModalInstance.close();
                    },function(){ });
                }else{
                	$uibModalInstance.close();
                }
            };
        }
    ]);

    app.controllerProvider.register('ScaleContainerController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            var currentNum = model.component.app.instances;
            $scope.container = { "operationType": "increase", "increaseNum": 1, "decreaseNum": 1, "currentNum": currentNum, "component": model.component.app.id};
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res,
                    "data": $scope.container
                });
            };
        }
    ]);
    app.controllerProvider.register('EditComponentController', ['$scope', '$uibModalInstance','$state','$localStorage','$q','ComponentService','NetworkService','CommonService','ServiceService','model',
        function($scope, $uibModalInstance, $state, $localStorage,$q,ComponentService,NetworkService,CommonService,ServiceService,model) {

            $scope.stepObj = { "previous": "", "current": "basic", "next": "network" };
            $scope.goNext = function(current) {
                $scope.stepObj = ComponentService.goNext(current);
            };
            $scope.goPrevious = function(current) {
                $scope.stepObj = ComponentService.goPrevious(current);
            };
            $scope.setCurrent = function(current) {
                $scope.stepObj = ComponentService.setCurrent(current);
            };
            $scope.component = angular.copy(model.component);
            // MESOS container doesn't have network property
            // so we made a fake one for form validation
            if ($scope.component.app.container.type === "MESOS") {
              $scope.component.app.container.docker.network = "HOST";
            }
            // init healthChecks portType, @TODO: We need it just because marathon will not save portType
            for ( var i = 0 ; i < $scope.component.app.healthChecks.length ; i++ )
                if ( $scope.component.app.healthChecks[i].port != undefined )
                    $scope.component.app.healthChecks[i].portType = 'PORT_NUMBER';
                else
                    $scope.component.app.healthChecks[i].portType = 'PORT_INDEX';
            // init uris and acceptedResourceRoles
            $scope.component.app.uris = $scope.component.app.uris ? $scope.component.app.uris.join() : '';
            $scope.component.app.acceptedResourceRoles = $scope.component.app.acceptedResourceRoles ? $scope.component.app.acceptedResourceRoles.join() : '';

            $scope.advance = { "isAlarm": false };
            $scope.alarmObj = { "upperLimit": "", "lowerLimit": "", "cpuUpperLimit": "","cpuLowerLimit":"", "memoryUpperLimit": "","memoryLowerLimit":"","SCALED_APP_ID":"","SCALE_NUMBER":"" };

            ServiceService.getapps(CommonService.getEndPoint(),$scope.component.appset_name).then(function(data){
                $scope.ScaledAppIds=data.data;
                if($scope.ScaledAppIds.indexOf($scope.componentJSON.marathon_app.id)==-1){
                    $scope.alarmObj.SCALED_APP_ID=$scope.ScaledAppIds[0];
                }else{
                    $scope.alarmObj.SCALED_APP_ID=$scope.ScaledAppIds[$scope.ScaledAppIds.indexOf($scope.componentJSON.marathon_app.id)]
                }
            })
            var componentJSON_1 = ComponentService.componentJSON().JSONTemplate;
            var portMapping_1 = ComponentService.componentJSON().portMapping;
            var volume_1 = ComponentService.componentJSON().volume;
			var ExternalAccess_1=angular.copy(portMapping_1);
			ExternalAccess_1.hostPort=0;
			$scope.ExternalAccess={
				checked:false,
				array:[angular.copy(ExternalAccess_1)]
			}
			$scope.$storage = $localStorage;
            $scope.componentJSON = angular.copy(componentJSON_1);
            $scope.componentJSON.marathon_app.id = $scope.component.app.id;
            $scope.componentRequest = {"appset_name": $state.params.serviceName, "app": $scope.componentJSON.marathon_app };

            $scope.advance = { "isAlarm": false };
            $scope.alarmObj = { "upperLimit": "", "lowerLimit": "", "cpuUpperLimit": "","cpuLowerLimit":"", "memoryUpperLimit": "","memoryLowerLimit":"" };

            $scope.envArray = {
            	'relationship':{},
            	'advanced':[]
            };

            $scope.paramsArray = [];
            $scope.labelArray = [];
            $scope.portArray = [];
            $scope.volumeArray = $scope.component.app.container.volumes||[];
            $scope.constraintArray = [];
            $scope.relationshipArray = [];
            $scope.argArray = [];
            $scope.UserDefinedNetwork=[];
            $scope.UDNetwork={
				'NetworkName':'',
				'NetworkDomain':''
			}

        	$scope.resetUDNetwork=function(){
        		$scope.UDNetwork={
					'NetworkName': (function(){
						if($scope.UserDefinedNetwork[0]&&$scope.UserDefinedNetwork[0].network){
							return $scope.UserDefinedNetwork[0].network.name
						}else return ""
					})(),
					'NetworkDomain':''
				}
        	};
        	$scope.resetExternalAccess=function(){
        		$scope.ExternalAccess.array = [angular.copy(ExternalAccess_1)];
        	}
        	NetworkService.getNetwork(CommonService.getEndPoint(), 0, 0, $scope.$storage.cluster._id)
        		.then(function(data){
        			_.each(data.data,function(item){
        				if(item.network.driver=="overlay")
        				$scope.UserDefinedNetwork.push(item);
        			})
        		});
        	$scope.changeDependency = function(){
        		_.each($scope.componentList,function(item){
        			if(item.app.id==$scope.dependency.selectedComponent){
        				$scope.dependency.servicePort=item.app.container.docker.portMappings||[];
        				$scope.dependency.show= _.some(item.app.labels,function(item){
            				return item=="linkermgmt"
            			})
        				$scope.dependency.domain = _.filter(item.app.container.docker.parameters,function(item){
		            		if(item.key=="net-alias"){
		            			return item.value
		            		}
		            	})||[];
        			}
        		})
        	}
        	$scope.setEnv = function() {
				if(!$scope.component.app.container.docker.network) {
					var temp = _.filter(model.components, function(item) {
						if($scope.dependency.selectedComponent == item.app.id) {
							return item
						}
					})[0];
					if(temp.app.container.docker.network=="BRIDGE") {
						return _.chain(temp.app.container.docker.parameters).map(function(item){
							if(item.key!='net'||(item.key == 'net' && item.value == $scope.UDNetwork.NetworkName)) {
								return true
							}else{
								return false
							}
						}).every()._wrapped;
					}
				}
				return false
			}

            var monitorKeyArray = ["ALERT_ENABLE","CPU_USAGE_HIGH_THRESHOLD","CPU_USAGE_LOW_THRESHOLD","INSTANCE_MAX_NUM","INSTANCE_MIN_NUM","MEMORY_USAGE_HIGH_THRESHOLD","MEMORY_USAGE_LOW_THRESHOLD", "SCALED_APP_ID","SCALE_NUMBER"];
            var customEnv = _.omit($scope.component.app.env, monitorKeyArray);
            var monitorEnv = _.pick($scope.component.app.env, monitorKeyArray);
            if(!_.isEqual(monitorEnv,{})){
                $scope.advance = { "isAlarm": true };
                $scope.alarmObj = { "upperLimit": monitorEnv.INSTANCE_MAX_NUM, "lowerLimit": monitorEnv.INSTANCE_MIN_NUM, "cpuUpperLimit": monitorEnv.CPU_USAGE_HIGH_THRESHOLD, "cpuLowerLimit": monitorEnv.CPU_USAGE_LOW_THRESHOLD,"memoryUpperLimit": monitorEnv.MEMORY_USAGE_HIGH_THRESHOLD,"memoryLowerLimit": monitorEnv.MEMORY_USAGE_LOW_THRESHOLD,"ALERT_ENABLE":true,"SCALE_NUMBER":monitorEnv.SCALE_NUMBER,"SCALED_APP_ID":monitorEnv.SCALED_APP_ID, "SCALE_NUMBER":monitorEnv.SCALE_NUMBER};
            }
            _.each($scope.component.app.container.docker.parameters,function(item){
              $scope.paramsArray.push({"key":item.key,"value":item.value});

              if(item.key === "net-alias"){
                $scope.component.app.container.docker.network = "";
                $scope.UDNetwork.NetworkDomain = item.value;
              } else if(item.key === "net"){
                $scope.UDNetwork.NetworkName = item.value;
              }
            });
            _.each($scope.component.app.container.docker.portMappings,function(item){
            	if(item.hostPort==0&&item.containerPort&&item.servicePort){
            		if($scope.ExternalAccess.checked){
            			$scope.ExternalAccess.array.push(item)
            		}else{
            			$scope.ExternalAccess.array=[item];
            			$scope.ExternalAccess.checked=true;
            		}
            	}else{
            		$scope.portArray.push(item);
            	}
            })
            _.each($scope.component.app.labels,function(value,key){
            	if(key!="HAPROXY_GROUP"){
               		$scope.labelArray.push({"key":key,"value":value});
            	}
            });
            _.each($scope.component.app.args,function(value){
                $scope.argArray.push({"value":value});
            });
            _.each($scope.component.app.constraints,function(value){
                $scope.constraintArray.push({ "key": value[0], "constraint": value[1], "value": value[2] });
            });
            $scope.component.app.cmd = $scope.component.app.cmd || "";
            $scope.relationshipArray = $scope.component.app.dependencies || [];

            $scope.componentList = model.components;
            $scope.dependency = {
            	"selectedComponent": (function(){
                    if($scope.relationshipArray[0]){
                        return $scope.relationshipArray[0]
                    }else if($scope.componentList[0]&&$scope.componentList[0].app){
            			return $scope.componentList[0].app.id||""
            		}else return ""
            	})(),
            	"servicePort":(function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app&&$scope.componentList[0].app.container&&$scope.componentList[0].app.container.docker){
            			return $scope.componentList[0].app.container.docker.portMappings||[]
            		}else return []
            	})(),
            	"show":(function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app&&$scope.componentList[0].app.labels){
            			return _.some($scope.componentList[0].app.labels,function(item){
            				return item=="linkermgmt"
            			})
            		}else return false
            	})(),
            	"domain":(function(){
            		if($scope.componentList[0]&&$scope.componentList[0].app&&$scope.componentList[0].app.container&&$scope.componentList[0].app.container.docker){
            			return _.filter($scope.componentList[0].app.container.docker.parameters,function(item){
			            		if(item.key=="net-alias"){
			            			return item.value
			            		}
			            	})||[]
            		}else return []
            	})()
            };
			$scope.clearUDNetArray=function(){
            	$scope.componentList.forEach(function(item){
	            	$scope.envArray.relationship[item.app.id]=new Array();
	           	})
            };
            $scope.clearUDNetArray();
            _.each(customEnv,function(value,key){
            	var flag=true;
            	_.each($scope.componentList,function(component){
            		if(value.indexOf('marathon-lb')!=-1&&value.indexOf(component.app.id)!=-1){
            			$scope.envArray.relationship[component.app.id].push({"key":key,"value":value});
            			flag=false;
            		}
            		if(component.app.container.docker.parameters&&component.app.container.docker.portMappings){
            			_.each(component.app.container.docker.parameters,function(item){
		            		if(item.key=="net-alias"&&value.indexOf(item.value)!=-1){
		            			_.each(component.app.container.docker.portMappings,function(port){
			        				if(value.indexOf(port.servicePort)){
			        					$scope.envArray.relationship[component.app.id].push({"key":key,"value":value})
			        					flag=false;
			        				}
			        			})
		            		}
	            		});
            		}
            	});
            	if(flag){
            		$scope.envArray.advanced.push({"key":key,"value":value});
            	}
            });


            $scope.addParameter = function(type) {
                switch (type) {
                    case "port":
                        $scope.portArray.push(angular.copy(portMapping_1));
                        break;
                    case "envRelation":
                        $scope.envArray.relationship[arguments[1]].push({ "key": "", "value": "" });
                        break;
                    case "envAdvanced":
                        $scope.envArray.advanced.push({ "key": "", "value": "" });
                        break;
                    case "parameters":
                        $scope.paramsArray.push({ "key": "", "value": "" });
                        break;
                    case "volume":
                        $scope.volumeArray.push(angular.copy(volume_1));
                        break;
                    case "externalaccess":
                    	$scope.ExternalAccess.array.push(angular.copy(ExternalAccess_1));
                    	break;
                    case "constraint":
                        $scope.constraintArray.push({ "key": "", "constraint": "UNIQUE", "value": "" });
                        break;
                    case "relationship":
                        if($scope.relationshipArray.indexOf($scope.dependency.selectedComponent)==-1){
							 $scope.relationshipArray.push($scope.dependency.selectedComponent);
                    	}
                        break;
                    case "arg":
                        $scope.argArray.push({ "value": "" });
                        break;
                    case "label":
                        $scope.labelArray.push({ "key": "", "value": "" });
                        break;
                }
            };
            $scope.removeParameter = function(type, index) {
                switch (type) {
                    case "port":
                        $scope.portArray.splice(index, 1);
                        break;
                    case "envRelation":
                        $scope.envArray.relationship[arguments[2]].splice(index,1);
                        break;
                    case "envAdvanced":
                        $scope.envArray.advanced.splice(index,1);
                        break;
                    case "parameters":
                        $scope.paramsArray.splice(index, 1);
                        break;
                    case "volume":
                        $scope.volumeArray.splice(index, 1);
                        break;
                    case "externalaccess":
                    	$scope.ExternalAccess.array.splice(index,1);
                    	break;
                    case "constraint":
                        $scope.constraintArray.splice(index, 1);
                        break;
                    case "relationship":
                       	$scope.relationshipArray.splice(index, 1);
                        break;
                    case "arg":
                        $scope.argArray.splice(index, 1);
                        break;
                    case "label":
                        $scope.labelArray.splice(index, 1);
                        break;
                }
            };

            $scope.addHealthCheck = function() {
                if ( !$scope.component.app.healthChecks )
                    $scope.component.app.healthChecks = [];
                $scope.component.app.healthChecks.push(angular.copy( ComponentService.componentJSON().HTTP ));
                // console.log( $scope.component.app.healthChecks );
            }

            $scope.removeHealCheck = function( index ) {
                if ( $scope.component.app.healthChecks && index > -1 && index < $scope.component.app.healthChecks.length )
                    $scope.component.app.healthChecks.splice( index, 1 );
            }

            $scope.changeHealthCheck = function( index, protocol ) {
                $scope.component.app.healthChecks[index] = angular.copy( ComponentService.componentJSON()[protocol] );
                // console.log($scope.component.app.healthChecks[index]);
            }

            $scope.changePortType = function( item ) {
                if ( item.portType == 'PORT_NUMBER' ) {
                    delete item.portIndex;
                    item.port = 0;
                } // if
                else {
                    delete item.port;
                    item.portIndex = 0;
                }
            }

            $scope.changeRuntime = function() {
                $scope.component.app.container.docker.network = "HOST";
                $scope.resetUDNetwork();
                $scope.clearUDNetArray();
                // Docker doesn't support gpu
                if ($scope.component.app.container.type == 'DOCKER') {
                    $scope.component.app.gpus = "";
                }
            }

            $scope.assembleData = function() {
                $scope.componentJSON.marathon_app.container.docker.portMappings = $scope.portArray;
                if($scope.ExternalAccess.checked){
                	$scope.componentJSON.marathon_app.container.docker.portMappings=$scope.componentJSON.marathon_app.container.docker.portMappings.concat($scope.ExternalAccess.array)
                }
                $scope.componentJSON.marathon_app.container.volumes = _.filter($scope.volumeArray,function(item){
                   return item.containerPath != "" && item.hostPath != "";
                });

                if ($scope.advance.isAlarm) {
                    var tempArray = [{ "key": "INSTANCE_MAX_NUM", "value": $scope.alarmObj.upperLimit },
                        { "key": "INSTANCE_MIN_NUM", "value": $scope.alarmObj.lowerLimit },
                        { "key": "CPU_USAGE_HIGH_THRESHOLD", "value": $scope.alarmObj.cpuUpperLimit },
                        { "key": "MEMORY_USAGE_HIGH_THRESHOLD", "value": $scope.alarmObj.memoryUpperLimit },
                        { "key": "CPU_USAGE_LOW_THRESHOLD", "value": $scope.alarmObj.cpuLowerLimit },
                        { "key": "MEMORY_USAGE_LOW_THRESHOLD", "value": $scope.alarmObj.memoryLowerLimit },
                        { "key": "SCALED_APP_ID", "value": $scope.alarmObj.SCALED_APP_ID},
                        { "key": "SCALE_NUMBER", "value": $scope.alarmObj.SCALE_NUMBER},
                        { "key": "ALERT_ENABLE", "value": "true" }
                    ];
                    $scope.envArray.advanced = $scope.envArray.advanced.concat(tempArray);
                }

                var tempEnvArray = _.filter($scope.envArray.advanced,function(item){return item.key != "";});
                for(var i in $scope.envArray.relationship){
                	tempEnvArray=tempEnvArray.concat( _.filter($scope.envArray.relationship[i],function(item){return item.key != "";}))
                }
                var envKeys = _.map(tempEnvArray, 'key');
                var envValues = _.map(tempEnvArray, 'value');
                $scope.componentJSON.marathon_app.env = ComponentService.handleSpecialItem(_.zipObject(envKeys, envValues));

                _.each($scope.paramsArray,function(item){
                     var tempKey = item.key;
                     if(tempKey != ""){
                        $scope.componentJSON.marathon_app.container.docker.parameters.push(item);
                     }
                });
                if(!$scope.component.app.container.docker.network){
                	$scope.componentJSON.marathon_app.container.docker.parameters.push(
                	{
                		"key":"net",
                		"value":$scope.UDNetwork.NetworkName,
                		"description":""
                	},{
                		"key":"net-alias",
                		"value":$scope.UDNetwork.NetworkDomain,
                		"description":""
                	});
                	$scope.component.app.container.docker.network="BRIDGE"
                };
                var tempLabelArray = _.filter($scope.labelArray,function(item){return item.key != "";});
                if($scope.ExternalAccess&&$scope.ExternalAccess.checked){
                	tempLabelArray.push({ "key": "HAPROXY_GROUP", "value": "linkermgmt" })
                }
                var labelKeys = _.map(tempLabelArray, 'key');
                var labelValues = _.map(tempLabelArray, 'value');
                $scope.componentJSON.marathon_app.labels = _.zipObject(labelKeys, labelValues);
                var finalConstraintArray = [];
                _.each($scope.constraintArray, function(constraint) {
                    var tempArray = [];
                    if (constraint.key != "") {
                        tempArray.push(constraint.key);
                        tempArray.push(constraint.constraint);
                         if(constraint.value != ""){
                             tempArray.push(constraint.value);
                         }
                        finalConstraintArray.push(tempArray);
                    }

                })
                $scope.componentJSON.marathon_app.container.docker.forcePullImage=$scope.component.app.container.docker.forcePullImage;
                $scope.componentJSON.marathon_app.container.docker.privileged=$scope.component.app.container.docker.privileged;
                $scope.componentJSON.marathon_app.container.docker.image = $scope.component.app.container.docker.image;
                $scope.componentJSON.marathon_app.instances = $scope.component.app.instances;
                $scope.componentJSON.marathon_app.cpus = $scope.component.app.cpus;
                $scope.componentJSON.marathon_app.gpus = $scope.component.app.gpus;
                $scope.componentJSON.marathon_app.mem = $scope.component.app.mem;
                $scope.componentJSON.marathon_app.disk = $scope.component.app.disk;
                $scope.componentJSON.marathon_app.container.type = $scope.component.app.container.type;
                $scope.componentJSON.marathon_app.container.docker.network = $scope.component.app.container.docker.network;
                $scope.componentJSON.marathon_app.cmd = ComponentService.handleSpecialItem($scope.component.app.cmd);
                $scope.componentJSON.marathon_app.constraints = finalConstraintArray;
                $scope.componentJSON.marathon_app.dependencies = $scope.relationshipArray;
                $scope.componentJSON.marathon_app.args = ComponentService.handleSpecialItem(_.without(_.map($scope.argArray, 'value'),""));
                $scope.componentJSON.marathon_app.uris = $scope.component.app.uris.length > 0 ? $scope.component.app.uris.split(',') : [];
                $scope.componentJSON.marathon_app.executor = $scope.component.app.executor;
                $scope.componentJSON.marathon_app.acceptedResourceRoles = $scope.component.app.acceptedResourceRoles.length > 0 ? $scope.component.app.acceptedResourceRoles.split(',') : [];
                $scope.componentJSON.marathon_app.healthChecks = $scope.component.app.healthChecks;
            }
            $scope.close = function(res) {
                if (res === "execute") {
                    $scope.assembleData();
                    model.saveComponent($scope.componentRequest, "edit").then(function(){
                    	$uibModalInstance.close();
                    },function(){});
                }else{
                	$uibModalInstance.close();
                }
            };
        }
    ]);
    app.controllerProvider.register('EditJSONController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.serviceCopy = angular.copy(model.service);
            $scope.service = {
                "name": $scope.serviceCopy.name,
                "description": $scope.serviceCopy.description,
                "group": JSON.stringify($scope.serviceCopy.group)
            };
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res,
                    "data": $scope.service
                });
            };
        }
    ]);
    app.controllerProvider.register('ContainerController', ['$scope', '$uibModalInstance', 'ContainerService', 'CommonService', 'ResponseService','model',
        function($scope, $uibModalInstance,ContainerService,CommonService,ResponseService,model) {
            $scope.close = function(res) {
                $uibModalInstance.close();
            };

            $scope.containerId = model.containerId;
            $scope.slaveId = model.slaveId;
            $scope.container = {};
            $scope.getDetail = function() {
                if (CommonService.endPointAvailable()) {
                    ContainerService.get(CommonService.getEndPoint(), $scope.containerId,$scope.slaveId).then(function(data) {
                            $scope.container = data;
                            $scope.handleEnv($scope.container.Config.Env);
                            $scope.handleVolume($scope.container.Mounts);
                        },
                        function(error) {
                            ResponseService.errorResponse(error, "container.getFailed");
                            $scope.envArray = [];
                        })
                }
            };
            $scope.envArray = [];
            $scope.handleEnv = function(envArray){
                _.each(envArray,function(envString){
                    var temp=envString.split("=");
                    $scope.envArray.push({"key":temp[0],"value":temp[1]});
                });
            };
            $scope.envCondition = $scope.envArray.length == 0?true:false;
            $scope.volumeArray = [];
            $scope.handleVolume = function(volumeArray){
                _.each(volumeArray,function(volume){
                    $scope.volumeArray.push({"containerPath":volume.Source,"hostPath":volume.Destination,"mode":volume.RW?"rw":volume.Mode});
                });
            };
            $scope.volumeCondition = $scope.volumeArray.length == 0?true:false;
            
            var isFetching = false;

            /********** Files ***********/
            $scope.files = [];
            $scope.currentFolder = [];

            var getFolder = function (path) {
              isFetching = true;
              ContainerService.browseFolder(CommonService.getEndPoint(), $scope.slaveId, path)
              .then(function (result) {
                $scope.files= []
                $scope.files = result.map(function(item) {
                  item.name = _.last(item.path.split('/'));
                  // Date string format: Thu Oct 19 2017 18:22:46 GMT+0800 (CST)
                  var time = new Date(item.mtime).toString().split(" ");
                  item.mtime = time[1] + " " + time[2] + " " + time[4];
                  if (item.size < 1024) {
                    item.size += " B";
                  } else if (item.size <= 1024000) {
                    item.size = _.ceil(item.size / 1024, 2) + " KB";  
                  } else if (item.size <= 1024000000) {
                    item.size = _.ceil(item.size / 1024 / 1024, 2) + " MB";  
                  } else {
                    item.size = _.ceil(item.size / 1024 / 1024 / 1024, 2) + " GB";  
                  }
                  
                  return item;
                });
                isFetching = false;
              });
            };

            $scope.browseFolder = function (name, path) {
              if (name !== "") {
                $scope.currentFolder.push(name);
              }
              getFolder(path);
            };

            $scope.backToFolder = function (idx) {
              $scope.currentFolder.splice(idx);
              var path = _.get($scope.volumeArray, "0.containerPath") + "/" + $scope.currentFolder.join('/');
              getFolder(path);
            };

            $scope.downloadFile = function (path) {
              var url = "http://" + CommonService.getEndPoint() + "/agent/" + $scope.slaveId + "/files/download?path=" + path;
              window.open(url, "_self");
            };

            /********** Logs ************/
            $scope.log = {
              content: [],
              type: "stdout",
              offset: 0,
              length: 5000
            };
            $scope.getLogs = function () {
              var params = {
                type: $scope.log.type,
                offset: $scope.log.offset,
                length: $scope.log.length,
                volumePath: _.get($scope.volumeArray, "0.containerPath")
              };
              isFetching = true;
              ContainerService.getContainerLogs(CommonService.getEndPoint(), $scope.slaveId, params)
              .then(function (result) {
                $scope.log.content = $scope.log.content.concat(result.data.split("\n"));
                $scope.log.offset += result.data.length;
                isFetching = false;
              });
            };

            $scope.changeLogType = function () {
              $scope.log.content = [];
              $scope.log.offset = 0;
              $scope.getLogs();
            };

            $scope.downloadLogs = function (type) {
              var volumePath = _.get($scope.volumeArray, "0.containerPath");
              var url = "http://" + CommonService.getEndPoint() + "/agent/" + $scope.slaveId + "/files/download?path=" + volumePath + "/" + $scope.log.type;
              window.open(url, "_self");
            };

            setTimeout(function() {
              var logArea = angular.element(document.getElementById("logs"));
              logArea.on("scroll", function() {
                var OFFSET = 50;
                // Hit the bottom
                if(!isFetching && logArea[0].scrollTop + logArea[0].clientHeight + OFFSET >= logArea[0].scrollHeight) {
                  $scope.getLogs();
                }
              });
            }, 1000);
        }
    ]);
});
