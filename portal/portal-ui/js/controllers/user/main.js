define(['app', 'services/user/main', 'services/cluster/main', 'directives/main', 'directives/pagination'], function(app) {
    'use strict';
    app.controllerProvider.register('UserController', ['$scope', '$uibModal', '$location', '$state', 'UserService', 'CommonService', 'ResponseService', 'ClusterService', function($scope, $uibModal, $location, $state, UserService, CommonService, ResponseService, ClusterService) {
        $scope.users = [];
        
        $scope.$watch('currentPage', function() {
            $scope.getUsers();
        });
        $scope.recordPerPage = CommonService.recordNumPerPage();
        $scope.totalPage = 1;
        $scope.currentPage = 1;      
        $scope.totalrecords = 0;
        $scope.getUsers = function() {
            var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
            var limit = $scope.recordPerPage;
            UserService.getUsers(skip, limit).then(function(data) {
                    $scope.totalrecords = data.count;
                    $scope.totalPage = Math.ceil($scope.totalrecords / $scope.recordPerPage);
                    $scope.users = data.data;
                },
                function(error) {
                    ResponseService.errorResponse(error, "user.listFailed");
                })
        };
        $scope.createUser = function() {
          $uibModal.open({
            templateUrl: 'templates/user/createUser.html',
            controller: 'CreateUserController',
            backdrop: 'static'
          })
          .result
          .then(function(response) {
            if (response.operation === 'execute') {
              $scope.saveUser(response.data, "create");
            }
          });
        };
        $scope.editUser = function(user) {
          user.roleType = user.rolename
          $uibModal.open({
            templateUrl: 'templates/user/editUser.html',
            controller: 'EditUserController',
            backdrop: 'static',
            resolve: {
              model: function() {
                return {
                  user: user
                };
              }
            }
          })
          .result
          .then(function(response) {
            if (response.operation === 'execute') {
              $scope.saveUser(response.data, "edit");
            }
          });
        };
        $scope.saveUser = function(user, type) {
          UserService.saveUser(user, type).then(function(data) {
            $scope.getUsers();
          },
            function(error) {
              ResponseService.errorResponse(error, "user.saveFailed");
          });
        };
        $scope.gotoClusterPage = function(user) {
            $location.path('/cluster/' + user.username);
        };
        $scope.checkCondition = function(user) {
            ClusterService.getClusters("","","unterminated",user._id).then(function(data) {
                    if (data.data.length == 0) {
                        $scope.confirmDelete(user);
                    } else {
                        $scope.absortDelete(user);
                    }
                },
                function(error) {
                    ResponseService.errorResponse(error, "user.getClusterFailed");
                })
        };

        $scope.absortDelete = function(user) {
            $uibModal.open({
                    templateUrl: 'templates/user/absortDelete.html',
                    controller: 'AbsortDeleteController',
                    backdrop: 'static',
                    resolve: {
                        model: function() {
                            return {
                                message: {
                                    "user": user,
                                    "title": "user.absortDeleteTitle",
                                    "content": "user.absortDeleteContent"
                                }
                            };
                        }
                    }
                })
                .result
                .then(function(response) {
                    if (response.operation === 'execute') {
                        // go to cluster page
                        // $location.path('/cluster/' + response.data.username);
                    }
                });
        };
        $scope.confirmDelete = function(user) {
            $scope.$translate(['common.deleteConfirm', 'user.deleteMessage', 'common.delete']).then(function(translations) {
                $scope.confirm = {
                    "title": translations['common.deleteConfirm'],
                    "message": translations['user.deleteMessage'],
                    "button": {
                        "text": translations['common.delete'],
                        "action": function() {
                            $scope.deleteUser(user);
                        }
                    }

                };
                CommonService.deleteConfirm($scope);
            });
        };
        $scope.deleteUser = function(user) {
            UserService.deleteUser(user).then(function(data) {
                    $scope.getUsers();
                },
                function(error) {
                    ResponseService.errorResponse(error, "user.deleteFailed");
                })
        };
       
        $scope.getUsers();
    }]);

    app.controllerProvider.register('CreateUserController', ['$scope', '$uibModalInstance',
        function($scope, $uibModalInstance, model) {
            $scope.user = {
                "username": "",
                "email": "",
                "roleType": "admin",
                "company": ""
            };
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res,
                    "data": $scope.user
                });
            };
        }
    ]);
    app.controllerProvider.register('EditUserController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.user = angular.copy(model.user);
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res,
                    "data": $scope.user
                });
            };
        }
    ]);
    app.controllerProvider.register('AbsortDeleteController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.user = angular.copy(model.message.user);
            $scope.message = {
                "title": model.message.title,
                "content": model.message.content
            };
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res,
                    "data": $scope.user
                });
            };
        }
    ]);

});
