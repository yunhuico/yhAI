define(['directives/module'], function(dirmodule) {
    'use strict';
    dirmodule.directive('loading', ['$http', '$timeout', function($http, $timeout) {
        return {
        restrict: 'ACEM',
        template: '<div style="position: fixed;top: 0px;right: 0px;left: 0px;bottom: 0px;background: rgba(0, 0, 0, 0.5);z-index: 10000;">'
            +'<div style="height: 100%;display: table;margin: auto;">'
            +'<div style="display: table-cell;vertical-align: middle;">'
            +'<img style="width:60px;height:60px;" src="img/loading.gif">'
            +'</div></div></div>',
        link: function(scope, elm, attrs) {
            scope.isLoading = function() {
                if ($http.pendingRequests.length > 0) {
                    var filter = /\.html$|\/clusterValidate|\/validate|\/nodemonitoring|\/containermonitoring|\/registryValidate|\/user\/login/;
                    for (var i in $http.pendingRequests) {
                        if (filter.test($http.pendingRequests[i].url)) {
                            return 0
                        }
                    }
                    return 1;
                } else return 0;
            };
            scope.$watch(scope.isLoading, function(v) {
                if (v) {
                    $(elm).show();
                } else {
                    $(elm).hide();
                }
            });
        }
      };
    }]);
});
