define(['app', 'services/service/main', 'directives/pagination', 'directives/status', 'directives/drag'], function(app) {
    'use strict';
    app.controllerProvider.register('ServiceController', ['$scope', '$uibModal', '$localStorage', 'ServiceService', 'CommonService', 'ResponseService', function($scope, $uibModal, $localStorage, ServiceService, CommonService, ResponseService) {

        $scope.$storage = $localStorage;
        $scope.services = [];
        $scope.recordPerPage = CommonService.recordNumPerPage();
        $scope.currentPage = 1;
        $scope.totalPage = 1;
        $scope.totalRecords = 0;
        $scope.selectedService = {};
        $scope.endPointAvailable = CommonService.endPointAvailable();
        $scope.$watch('currentPage', function(newValue, oldValue) {
            if (newValue != oldValue) {
                $scope.getServices();
            }
        });
        $scope.$watch('$storage.cluster', function(newValue, oldValue) {
            if (!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id) {
                $scope.currentPage = 1;
                $scope.totalPage = 1;
                $scope.totalRecords = 0;
                $scope.services = [];
                $scope.endPointAvailable = CommonService.endPointAvailable();
                $scope.getServices();
            }

        }, true);

        $scope.getServices = function() {
            if ($scope.endPointAvailable) {
                var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
                var limit = $scope.recordPerPage;
                ServiceService.get(CommonService.getEndPoint(), skip, limit).then(function(data) {
                        $scope.totalrecords = data.count;
                        $scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
                        $scope.services = data.data;
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.listFailed");
                    })
            } else {
                $scope.totalrecords = 0;
                $scope.totalPage = 0;
                $scope.services = [];
            }
        };
        $scope.createService = function() {
            if ($scope.endPointAvailable) {
                $uibModal.open({
                        templateUrl: 'templates/service/createService.html',
                        controller: 'CreateServiceController',
                        backdrop: 'static'
                    })
                    .result
                    .then(function(response) {
                        if (response.operation === 'execute') {
                            $scope.saveService(response.data);
                        }
                    });
            }
        };
        $scope.saveService = function(service) {
            if ($scope.endPointAvailable) {
                ServiceService.save(CommonService.getEndPoint(), service).then(function(data) {
                        $scope.getServices();
                    },
                    function(error) {
                    	ResponseService.errorResponse(error, "service.saveFailed");
                    })
            }
        };
        $scope.confirmDelete = function(service) {
            $scope.$translate(['common.deleteConfirm', 'service.deleteMessage', 'common.delete']).then(function(translations) {
                $scope.confirm = {
                    "title": translations['common.deleteConfirm'],
                    "message": translations['service.deleteMessage'],
                    "button": {
                        "text": translations['common.delete'],
                        "action": function() {
                            $scope.deleteService(service);
                        }
                    }

                };
                CommonService.deleteConfirm($scope);
            });
        };
        $scope.deleteService = function(service) {
            if ($scope.endPointAvailable) {
                ServiceService.delete(CommonService.getEndPoint(), service).then(function(data) {
                        $scope.getServices();
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.deleteFailed");
                    })
            }
        };
        $scope.stopService = function(service) {
            if ($scope.endPointAvailable) {
                ServiceService.stop(CommonService.getEndPoint(), service).then(function(data) {
                        $scope.getServices();
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.stopFailed");
                    })
            }
        };
        $scope.startService = function(service) {
            if ($scope.endPointAvailable) {
                ServiceService.start(CommonService.getEndPoint(), service).then(function(data) {
                        $scope.getServices();
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "service.startFailed");
                    })
            }
        };
        $scope.refresh = function() {
             $scope.currentPage = 1;
             $scope.totalPage = 1;
             $scope.totalRecords = 0;               
             $scope.getServices();
        };
        $scope.getServices();


    }]);

    app.controllerProvider.register('CreateServiceController', ['$scope', '$uibModalInstance',
        function($scope, $uibModalInstance) {
            $scope.service = {
                "name": "",
                "description": "",
                "group": "",
                "created_by_json": false
            };
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res,
                    "data": $scope.service
                });
            };
            $scope.showTypeWarning = false;
            $scope.showLengthWarning = false;
        }
    ]);




});
