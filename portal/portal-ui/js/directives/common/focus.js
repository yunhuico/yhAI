define(['directives/module'], function(dirmodule) {
    'use strict';
    dirmodule.directive('autofocus', ['$timeout', function($timeout) {
        return {
            restrict: 'ACEM',
            link: function(scope, elm, attrs, ctrl) {
                $timeout(function() {
                    elm[0].focus();
                });
            }
        };
    }]);
});
