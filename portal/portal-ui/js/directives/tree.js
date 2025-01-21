define(['app', 'directives/showMessage'], function(app) {
	'use strict';
	app.compileProvider.directive('tree', ['$compile', function($compile) {
		return {
			restrict: 'E',
			scope: {
				data: '=',
				itemName: '=',
				submitform:'='
			},	

			templateUrl:'templates/directive/tree.html',
			
			controller: ['$scope', function($scope) {
				$scope.isRequired = function(required, key) {
					var name = _.find(required, function(value) {
						return value === key;
					});
					return name ? true : false;
				};
			}],
			compile: function(tElement, tAttr, transclude) {
            		var contents = tElement.contents().remove();
            		var compiledContents;
            		return function(scope, iElement, iAttr) {
                		if(!compiledContents) {
                    		compiledContents = $compile(contents, transclude);
                		}
                		compiledContents(scope, function(clone, scope) {
                         iElement.append(clone); 
                		});
					
        			}
			}
		}
	}])
})