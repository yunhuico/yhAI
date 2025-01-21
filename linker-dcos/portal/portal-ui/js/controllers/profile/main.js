define(['app', 'services/user/main', 'directives/password'], function(app) {
    'use strict';
    app.controllerProvider.register('ProfileController', ['$scope', '$uibModal', '$location', '$cookies', 'UserService', 'CommonService', 'ResponseService', function($scope, $uibModal, $location, $cookies, UserService, CommonService, ResponseService) {
        $scope.passwords = { "password": "", "newpassword": "", "confirm_newpassword": "" };
        $scope.showPwdChange = false;
        $scope.getUserInfo = function() {
            $scope.showPwdChange = false;
            UserService.getUserInfo().then(function(data) {                    
                     $scope.profile = data.data;
                     $scope.profile.roleType = data.data.rolename;
                },
                function(error) {
                    ResponseService.errorResponse(error, "user.listFailed");
                });
        };

        $scope.clearPwd = function(){
            $scope.showPwdChange = true;
            $scope.passwords = { "password": "", "newpassword": "", "confirm_newpassword": "" };
        }

        $scope.updateUserInfo = function() {
            UserService.saveUser($scope.profile, "edit").then(function(data) {
                    $uibModal.open({
                        templateUrl: 'templates/common/success.html',
                        controller: 'SuccessController',
                        size: "sm",
                        resolve: {
                            model: function() {
                                return {
                                    "title": "user.updateSuccess",
                                    "message": "user.updateSuccess",
                                    'button': {
                                        'text': 'common.close'                                      
                                    },
                                };
                            }
                        }
                    });
                },
                function(error) {
                    ResponseService.errorResponse(error);
                });
        };
        $scope.updatePassword = function() {
            UserService.updatePasssword($scope.passwords).then(function(data) {
                    $uibModal.open({
                            templateUrl: 'templates/common/success.html',
                            controller: 'SuccessController',
                            size: "sm",
                            resolve: {
                                model: function() {
                                    return {
                                        "title": "user.updatePasswordSuccess",
                                        "message": "user.updatePasswordSuccess",
                                        'button': {
                                            'text': 'common.close'                                      
                                         },
                                    };
                                }
                            }
                        })
                        .result
                        .then(function(response) {
                            CommonService.logOut().then(function(response) {
                                    $cookies.remove('username');
                                    window.location = "/login.html";
                                },
                                function(errorMessage) {
                                    ResponseService.errorResponse(errorMessage);
                                });

                        });

                },
                function(error) {
                    ResponseService.errorResponse(error);
                });
        };





    }]);



});
