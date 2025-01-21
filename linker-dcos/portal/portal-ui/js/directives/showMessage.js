define(['app'], function(app) {
	'use strict';
	app.compileProvider.directive('showMessage', function() {
		return {
			restrict: 'E',
			scope: {
				isShow: '='
			},
			template:'<a ng-click="setShowAndHide()"><span class="glyphicon glyphicon-info-sign" style="color:black;margin-left:15px;"></span></a>',
			link: function(scope, ele, attr) {
				scope.setShowAndHide = function() {
					scope.isShow = !scope.isShow;	
				}
			}
		}
	});
	
	app.compileProvider.directive('integer', function() {
		return {
			restrict:"A",
			require:'ngModel',
 			link: function (scope, iElement, iAttrs,ngController) {
 				var reg = /^-?\d+$/;
 				scope.$watch(iAttrs.ngModel, function (newVal) {
 					if(!newVal) return;
					if(!reg.test(newVal)){
 						ngController.$setValidity('integer', false);
					}else{
				 		ngController.$setValidity('integer', true);
					}
				});
			}
 		}
	});

	app.compileProvider.directive('regex', function() {
		return {
			restrict: 'A',
			require: 'ngModel',
			link: function(scope, iElement, iAttrs, ngController) {
				var reg_string = iAttrs.regex;
				if(reg_string) {
					var reg = new RegExp(reg_string);
					scope.$watch(iAttrs.ngModel, function(newVal) {
						if(!newVal) return;
						if(!reg.test(newVal)){
	 						ngController.$setValidity('regex', false);
						}else{
					 		ngController.$setValidity('regex', true);
						}
					})
				}
			}
		}
	})
})