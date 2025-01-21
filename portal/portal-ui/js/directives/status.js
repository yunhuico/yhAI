define(['app'], function(app) {
    'use strict';
    app.compileProvider.directive('status', function() {
        return {
            restrict: 'E',
            scope: {
              type: '@'             
            },
            template: '<span ng-class="{label:true, \
                \'label-success\': type==\'RUNNING\'||type==\'success\'|| type==\'TASK_RUNNING\'|| type==\'Healthy\',\
                \'label-danger\': type==\'FAILED\' || type==\'OFFLINE\'|| type==\'STOPPED\'|| type==\'SUSPENDED\'|| type==\'fail\' || type==\'DELETING\'|| type==\'UnHealthy\',\
                \'label-warning\': type==\'TERMINATING\' || type==\'DEPLOYING\' || type==\'INSTALLING\'|| type==\'INCOMPLETE\'|| type==\'start\' || type==\'WAITING\',\
                \'label-info\': type==\'IDLE\',\'label-default\': type==\'UNKNOWN\' || type==\'\'}">\
                {{\'common.\' + (type || \'unknown\') | translate}}</span>',
            replace: true      
        }
    });
});
