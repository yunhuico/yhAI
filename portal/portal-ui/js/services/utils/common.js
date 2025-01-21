define(['services/module'], function (app) {
    'use strict';
     app.factory('CommonService', ['$http','$q','$translate','$uibModal','$localStorage',function ($http, $q, $translate,$uibModal,$localStorage) {
            return {
            	logOut:function(){
            		var url = "/logout";
					var request = {
						"url": url,
						"method": "GET"
					}
					var deferred = $q.defer();
					$http(request).success(function(response) {
						deferred.resolve(response);
					}).error(function(error) {
						deferred.reject(error);
					});
					return deferred.promise;
            	},
				getSupportedLangs : function(){
					var supportedLangs =  [{
		          			"name" :  "en",
		          			"display" : "English"        
				    },{
				          "name" :  "zh",
				          "display" : "中文" 
				    }];
					return supportedLangs;
				},
				getCurrentLang : function(){	
				   var self = this;				    
			       var storage = $translate.storage();
			       var key = $translate.storageKey();
			       var lang = storage.get(key) || (window.navigator.language.indexOf("zh")!=-1 ? "zh" : "en");  
			       $translate.use(lang);
			       var currentLang =  _.find(self.getSupportedLangs(),function(langObj){ 
				           return langObj.name == lang;           
				   });   
			       return currentLang;
			    },
			    setLang : function(lang){	
				   $translate.use(lang);
			    },
                deleteConfirm : function($scope){
					$uibModal.open({
					    templateUrl: 'templates/common/confirm.html',
					    controller: 'ConfirmController',
					    size: 'sm',
					    backdrop:'static',
					    resolve: {
					        model: function () {
					            return $scope.confirm;
					        }
					    }
				   });
				},
				recordNumPerPage:function(){
					var param = 20;
					return param;
				},
				clusterAvailable : function(){
					if(!angular.isUndefined($localStorage.cluster) && !_.isEmpty($localStorage.cluster)){
                       return true;
					}
					return false;
				},
				endPointAvailable : function(){
					if(this.clusterAvailable() && !angular.isUndefined($localStorage.cluster.endPoint) && !_.isEmpty($localStorage.cluster.endPoint)){
                       return true;
					}
					return false;
					// return true;
				},
				getEndPoint: function(){
					return $localStorage.cluster.endPoint;
					// return "192.168.5.237:10004";
				},
				generateContainerName : function(slaveid,name){
                     return "mesos-"+slaveid+"."+name;
				},
				getHostEndPoint: function (){
					return window.location.hostname;
				}





	    	}
	    	
     }]);
});