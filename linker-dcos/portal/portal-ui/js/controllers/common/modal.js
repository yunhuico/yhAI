define(['controllers/module'], function(ctlmodule) {
    'use strict';
    ctlmodule.controller('ConfirmController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.title = model.title;
            $scope.message = model.message;
            $scope.button = model.button;
            $scope.close = function(res) {
                $uibModalInstance.close(res);
            };
            $scope.doAction = function(res, action) {
                if ($scope.button.argu) {
                    action($scope.button.argu);
                } else {
                    action();
                }
                $uibModalInstance.close(res);
            };
        }
    ]);
    ctlmodule.controller('ConfirmAbsortController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.message = model.message;
            $scope.close = function(res) {
                $uibModalInstance.close({
                    "operation": res
                });
            };
        }
    ]);
    ctlmodule.controller('ActionFailedController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.message = model.message;
            $scope.detailStatus = false;
            $scope.toggleDetail = function(){
                $scope.detailStatus = !$scope.detailStatus;
            };
            $scope.close = function(res) {
                $uibModalInstance.close(res);
            };
        }
    ]);
    ctlmodule.controller('CommonController', ['$scope', '$cookies', '$window','$localStorage', '$http', 'CommonService', 'ResponseService',
        function($scope, $cookies, $window,$localStorage, $http,CommonService, ResponseService) {
            var url = "/user/profile";
            var request = {
                "url": url,
                "method": "GET"
            }

            $http(request).success(function(data) {
              $scope.rolename = data.data.rolename;
            }).error(function(error) {
                ResponseService.errorResponse(error);
                $scope.logOut()
            });
            $scope.$storage = $localStorage;
            $scope.currentUser = $cookies.get('username');
            $scope.currentLangName = CommonService.getCurrentLang().name;
            $scope.changeLang = function(lang) {
                $scope.currentLangName = lang;
                CommonService.setLang(lang);
            };

            $scope.logOut = function() {
                CommonService.logOut().then(function(response) {
                        $cookies.remove('username');
                        $localStorage.$reset();
                        window.location = "/login.html";
                    },
                    function(errorMessage) {
                        ResponseService.errorResponse(errorMessage);
                    });
            }
        }
    ]);
    ctlmodule.controller('SuccessController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.title = model.title;
            $scope.message = model.message;
            $scope.button = model.button;
            $scope.close = function(res) {
                $uibModalInstance.close(res);
            };
            $scope.doAction = function(res, action) {
                if(action){
                     if ($scope.button.argu) {
                        action($scope.button.argu);
                     } else {
                        action();
                     }
                }

                $uibModalInstance.close(res);
            };
        }
    ]);
	ctlmodule.controller('PromptController', ['$scope', '$uibModalInstance', 'model',
        function($scope, $uibModalInstance, model) {
            $scope.message = model;
            $scope.detailStatus = false;
            $scope.toggleDetail = function(){
                $scope.detailStatus = !$scope.detailStatus;
            };
            $scope.button = model.button;

            $scope.close = function(res) {
                $uibModalInstance.close(res);
            };
            $scope.doAction = function(res, action) {
                if(action){
                     if ($scope.button.argu) {
                        action($scope.button.argu);
                     } else {
                        action();
                     }
                }

                $uibModalInstance.close(res);
            };
        }
    ]);

});
