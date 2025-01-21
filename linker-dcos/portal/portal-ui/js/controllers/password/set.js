define(['login', 'directives/newpassword', 'services/password/set'], function(app) {
    'use strict';
    app.controllerProvider.register('SetPasswordController', ['$scope', '$location', '$window', '$uibModal', 'SetPasswordService', 'ResponseService', function($scope, $location, $window, $uibModal, SetPasswordService, ResponseService) {
		$scope.passwords = {
			newpassword: '',
			confirmpassword: ''
		}
		$scope.username = $location.search().username;
		$scope.resetPassword = function() {
			SetPasswordService.resetPassword($scope.passwords, $scope.username, $location.search().activecode).then(function(data) {
				$uibModal.open({
					templateUrl: 'templates/common/success.html',
					controller: 'SuccessController',
					size: 'sm',
					backdrop: 'static',
					resolve: {
						model: function() {
							return {
								'title': 'user.updateSuccess',
								'message': 'user.updateSuccess',
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
				ResponseService.errorResponse(error,"user.loginFailed");
			})
		}
    }]);



});
