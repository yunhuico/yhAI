define(['app','services/log/main','directives/pagination','directives/status'], function(app) {
    'use strict';
    app.controllerProvider.register('LogController', ['$scope', '$localStorage','CommonService', 'ResponseService','LogService', function($scope, $localStorage,CommonService,ResponseService,LogService) {
        $scope.$storage = $localStorage;

        $scope.recordPerPage = CommonService.recordNumPerPage();
        $scope.currentPage = 1;  
        $scope.totalPage = 1;       
        $scope.totalRecords = 0;  
        $scope.logs = [];          
        $scope.$watch('currentPage', function(newValue,oldValue) {
            if (newValue != oldValue) {
            	$scope.getLogs();
            }	            
        });
        $scope.$watch('$storage.cluster', function(newValue,oldValue) {
			if(!_.isUndefined(newValue) && !_.isUndefined(oldValue) && newValue._id != oldValue._id){
				 $scope.currentPage = 1;  
                 $scope.totalPage = 1;       
                 $scope.totalRecords = 0;  
                 $scope.logs = [];     
                 $scope.getLogs();
			}
			
		},true);
		
        $scope.getLogs = function() {
            if (CommonService.clusterAvailable()) {
            	var skip = ($scope.currentPage - 1) * $scope.recordPerPage;
				var limit = $scope.recordPerPage;
                LogService.getLogs($scope.$storage.cluster._id,skip, limit).then(function(data) {
                        $scope.totalRecords = data.count;
					    $scope.totalPage = Math.ceil($scope.totalRecords / $scope.recordPerPage);
                        $scope.logs = data.data;
                    },
                    function(error) {
                        ResponseService.errorResponse(error, "log.listFailed");
                    })
            }else{
                 $scope.totalRecords = 0;
                 $scope.totalPage = 0;
                 $scope.logs = [];
            }
        };
        $scope.refresh = function(){
           $scope.currentPage = 1;  
           $scope.totalPage = 1;       
           $scope.totalRecords = 0;       
           $scope.getLogs();
        };
        $scope.getLogs();

    }]);

   

});
