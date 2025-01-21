define(['app','services/component/main','directives/status'], function(app) {
  'use strict';
  app.controllerProvider.register('ComponentController', ['$scope', '$localStorage', 'ResponseService', 'ComponentService', function($scope, $localStorage,ResponseService,ComponentService) {
    $scope.$storage = $localStorage;
    $scope.components = [];

    $scope.$watch('$storage.cluster', function(newValue,oldValue) {
      if(!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id){
        $scope.getComponents();
      }
    },true);

    $scope.getComponents = function () {
      ComponentService.getComponents($scope.$storage.cluster._id)
      .then(function(result) {
        $scope.components = result.data.componentStatus;
      }, function(error) {
        ResponseService.errorResponse(error, "component.getComponentFail");
      });
    };

    $scope.getComponents();
  }]);
});
