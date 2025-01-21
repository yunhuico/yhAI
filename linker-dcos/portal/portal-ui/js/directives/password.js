define(['app'], function(app) {
    'use strict';
    app.compileProvider.directive('oldpassword', ["$q", "$http","$cookies", function($q, $http,$cookies) {
        return {
            restrict: 'ACEM',
            require: 'ngModel',
            link: function(scope, elm, attrs, ctrl) {

                ctrl.$asyncValidators.oldpassword = function(modelValue, viewValue) {

                    if (ctrl.$isEmpty(modelValue)) {
                        // consider empty model valid
                        return $q.when();
                    }
                    var deferred = $q.defer();
                    var url = "/user/login";
                    var request = {
                        "url": url,
                        "dataType": "json",
                        "method": "POST",
                        "data": JSON.stringify({ "username": $cookies.get('username'), "password": viewValue })
                    }
                    $http(request).success(function(response) {
                        deferred.resolve(response);
                    }).error(function(error) {
                        deferred.reject(error);
                    });
                    return deferred.promise;

                };
            }
        };
    }]);
    app.compileProvider.directive('confirmpassword', ["$q", "$http", function($q, $http) {
        return {
            restrict: 'ACEM',
            require: 'ngModel',
            link: function(scope, elm, attrs, ctrl) {
                 elm.on('blur keyup change', function() {          
                    var newpassword = scope.$parent.$parent.passwords.newpassword;
                    if (ctrl.$viewValue === newpassword) {                
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
