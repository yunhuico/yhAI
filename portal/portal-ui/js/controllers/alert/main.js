define(['app','services/alert/main','directives/pagination','directives/status'], function(app) {
    'use strict';
    app.controllerProvider.register('AlertController', ['$scope', '$localStorage','CommonService', 'ResponseService','AlertService', function($scope, $localStorage,CommonService,ResponseService,AlertService) {
      $scope.$storage = $localStorage;

      $scope.recordPerPage = CommonService.recordNumPerPage();
      $scope.currentPage = 1;
      $scope.totalPage = 1;
      $scope.totalRecords = 0;
      $scope.alerts = [];
      var alertName = "", alertAction = "";

      $scope.$watch('currentPage', function(newValue,oldValue) {
        if (newValue != oldValue) {
          getAlerts();
        }             
      });
      $scope.$watch('$storage.cluster', function(newValue,oldValue) {
        if(!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id){
          $scope.currentPage = 1;  
          $scope.totalPage = 1;       
          $scope.totalRecords = 0;  
          $scope.alerts = [];     
          getAlerts();
        }
      },true);

      var getAlerts = function() {
        if (CommonService.endPointAvailable()) {
          var params = {
            "skip": ($scope.currentPage - 1) * $scope.recordPerPage,
            "limit": $scope.recordPerPage,
            "alert_name": alertName || "",
            "action": alertAction || ""
          };
          AlertService.getAlerts(CommonService.getEndPoint(), params).then(function(data) {
              $scope.totalRecords = data.count;
              $scope.totalPage = Math.ceil($scope.totalRecords / $scope.recordPerPage);
              $scope.alerts = data.data;
            },
            function(error) {
              ResponseService.errorResponse(error, "alert.listFailed");
          });
        } else{
          $scope.totalRecords = 0;
          $scope.totalPage = 0;
          $scope.alerts = [];
        }
      };

      $scope.searchByName = function(val) {
        $scope.currentPage = 1;
        alertName = val;
        getAlerts();
      };

      $scope.searchByAction = function(val) {
        $scope.currentPage = 1;
        alertAction = val;
        getAlerts();
      };

      $scope.refresh = function() {
         $scope.currentPage = 1;  
         $scope.totalPage = 1;       
         $scope.totalRecords = 0;       
         getAlerts();
      };

      getAlerts();
    }]);
});
