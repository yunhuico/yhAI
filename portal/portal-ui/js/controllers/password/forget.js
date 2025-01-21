define(['login', 'services/password/forget'], function (app) {
    'use strict';
     app.controllerProvider.register('ForgetPasswordController', ['$scope', '$location', '$window', '$uibModal', 'ForgetPasswordService', 'ResponseService' ,function ($scope, $location, $window, $uibModal, ForgetPasswordService, ResponseService) {
    	 	$scope.username = '';
    	 	$scope.forgetPassword = function() {
    	 		ForgetPasswordService.forgetPassword($scope.username, $location.host(), $location.port()).then(function(data) {
    	 			$uibModal.open({
						templateUrl: 'templates/common/success.html',
						controller: 'SuccessController',
						size: 'sm',
						backdrop: 'static',
						resolve: {
							model: function() {
								return {
									'title': 'common.sendSuccess',
									'message': 'common.sendSuccess',
									'button': {
										'text': 'common.close'
									}
								}
							}
						}
					})
					.result
					.then(function(response) {
						$window.location = 'login.html';
					})

    	 		}, function(error) {
    	 			ResponseService.errorResponse(error);
    	 		});
    	 	}
     }]);


});
