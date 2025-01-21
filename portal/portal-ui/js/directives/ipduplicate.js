define(['app'], function(app) {
	'use strict';
	app.compileProvider.directive('ipduplicate', function() {
		return {
			restrict: 'A',
			require: 'ngModel',
			link: function(scope, ele, attrs, ngModelController) {
				if (!ngModelController) return;
				var index =attrs.name.substring(2,attrs.name.length)-1;
				ele.on('blur keyup change', function() {
					var ips = scope,
						path = attrs.ipduplicate.split('.');
					for (var i = 0; i < path.length; i++) {
						ips = ips[path[i]]
					}
					for (var i =0;i<ips.length;i++){
						if(ips[i].ip == ngModelController.$modelValue&&i!=index){
							ngModelController.$setValidity('duplicate', false);
							scope.$apply();
							return;
						}else{
							ngModelController.$setValidity('duplicate', true);
							scope.$apply();
						}
					}
				});
			}
		};
	});
});