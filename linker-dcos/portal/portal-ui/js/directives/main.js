define(['app'], function(app) {
    'use strict';
    app.compileProvider.directive('duplicate', ["$q", "$http", function($q, $http) {
        return {
            restrict: 'ACEM',
            require: 'ngModel',
            link: function(scope, elm, attrs, ctrl) {

                ctrl.$asyncValidators.duplicate = function(modelValue, viewValue) {

                    if (ctrl.$isEmpty(modelValue)) {
                        // consider empty model valid
                        return $q.when();
                    }

                    var deferred = $q.defer();
                    var url = "/user/validate";
                    var request = {
                        "url": url,
                        "dataType": "json",
                        "method": "GET",
                        "params": {
                            "username": viewValue
                        }
                    }

                    $http(request).success(function(response) {
                        deferred.resolve(response);
                    }).error(function(error) {
                        deferred.reject(error);
                    })


                    return deferred.promise;



                };
            }
        };
    }]);

});
