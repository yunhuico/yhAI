define(['login'], function(app) {
    'use strict';
    app.compileProvider.directive('confirmpassword', ["$q", "$http", function($q, $http) {
        return {
            restrict: 'ACEM',
            require: 'ngModel',
            link: function(scope, elm, attrs, ctrl) {
                 elm.on('blur keyup change', function() {          
                    var newpassword = scope.passwords.newpassword;
                    if (ctrl.$viewValue === newpassword || !ctrl.$viewValue) {                
                        ctrl.$setValidity('confirmpassword', true);
                        scope.$apply();
                        return;
                    }                 
                    ctrl.$setValidity('confirmpassword', false);
                    scope.$apply();
                    return;
                  });

            }
        };
    }]);



});