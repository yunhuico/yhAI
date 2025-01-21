define(['app'], function (app) {
    'use strict';
     app.provide.factory('FrameworkService', ['$http', '$q', '$uibModal',function ($http, $q, $uibModal) {
	    	return {
				get: function(clientAddr, type, name) {
					var deferred = $q.defer();
					if(_.isUndefined(name) || _.isEmpty(name)) {
						name = '';
					} else {
						name = '/' + name;
					}
					var url = "/package/" + type + name;
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
							"clientAddr": clientAddr
						}
					}
					$http(request).success(function(data) {
						deferred.resolve(data);
					}).error(function(error) {
						deferred.reject(error);
					});
					return deferred.promise;
				},
			    delete : function(clientAddr, framework, type){
                    var deferred = $q.defer();
					var url = "/frameworks/"+ type + "/" + framework.name;
					var request = {
						"url": url,
						"dataType": "json",
						"method": "DELETE",
						"params": {
						  "clientAddr": clientAddr
					    }
					}
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    install : function(clientAddr, name, options, version){
			    		var deferred = $q.defer();
					var url = "/package/install";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"data": angular.toJson({
			    				packageName: name,
			    				options: options,
			    				packageVersion: version
			    			}),
						"params": {
						  "clientAddr": clientAddr,
					    }
					}
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
			    },
			    advancedInstall: function(clientAddr, name, options, version) {
			    		var deferred = $q.defer();
			    		var url = "/package/install";
			    		var request = {
			    			"url": url,
			    			"dataType": "json",
			    			"method": "POST",
			    			"data": angular.toJson({
			    				packageName: name,
			    				options: options,
			    				packageVersion: version
			    			}),
			    			"params": {
			    				"clientAddr": clientAddr
			    			}
			    		}
			    		$http(request).success(function(data) {
			    			deferred.resolve(data);
			    		}).error(function(error) {
			    			deferred.reject(error);
			    		});
			    		return deferred.promise;
			    },
			 	uninstall: function(clientAddr, name, installedPackage) {
			 		var a = 0;
			 		var all = false;
			 		_.each(installedPackage, function(item) {
			 			if(item.packageInformation.packageDefinition.name == name) {
			 				a++;
			 			}
			 		});
			 		if(a > 1) {
			 			all = true;
			 		}
			 		var deferred = $q.defer();
			 		var url = "/package/uninstall";
			 		var request = {
			 			"url": url,
			 			"dataType": "json",
			 			"method": "POST",
			 			"data": angular.toJson({packageName:name, all:all}),
			 			"params": {
			 				"clientAddr": clientAddr
			 			}
			 		};
			 		$http(request).success(function(data) {
			 			deferred.resolve(data);
			 		}).error(function(error) {
			 			deferred.reject(error);
			 		});
			 		return deferred.promise;
			 	},
				getTasks :function(clientAddr,host_ip,skip, limit){
					var deferred = $q.defer();
					var url = "/tasks?count=true&skip="+skip+"&limit="+limit+"&host_ip="+host_ip;
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
						  "clientAddr": clientAddr
					    }
					}
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
				},

				getDescribe: function(clientAddr, type, name, version) {
					var deferred = $q.defer();
					var url = "/package/describe";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"data": angular.toJson({
			    				packageName: name,
			    				packageVersion: version
			    			}),
						"params": {
							"clientAddr": clientAddr
						}
					}
					$http(request).success(function(data){
						deferred.resolve(data);
					}).error(function(error){
						deferred.reject(error);
					});
					return deferred.promise;
				},

				getRepository: function(clientAddr) {
					var deferred = $q.defer();
					var url = "/package/repository/list";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "GET",
						"params": {
							"clientAddr": clientAddr,
						},
					}
					$http(request).success(function(data) {
						deferred.resolve(data);
					}).error(function(error) {
						deferred.reject(error);
					});
					return deferred.promise;
				},

				createRepository: function(clientAddr, data) {
					var deferred = $q.defer();
					var url = "/package/repository/add";
					var request = {
						"url": url,
						"dataType": "json",
						"method": "POST",
						"params": {
							"clientAddr": clientAddr
						},
						"data": angular.toJson(data)
					}
					$http(request).success(function(data) {
						deferred.resolve(data);
					}).error(function(error) {
						deferred.reject(error);
					});
					return deferred.promise;
				},
				deleteRepository: function(clientAddr, repo) {
					var deferred = $q.defer();
					var url = "/package/repository/" + repo.name;
					var request = {
						"url": url,
						"dataType": "JSON",
						"method": "DELETE",
						"params": {
							"clientAddr": clientAddr
						}
					}
					$http(request).success(function(data) {
						deferred.resolve(data);
					}).error(function(error) {
						deferred.reject(error);
					});
					return deferred.promise;
				},
				getShowJson: function(value) {
					var showJson = {};
				    var length = Object.keys(value).length;
				    for(var i=0; i<length; i++) {
				    var itemName = Object.keys(value)[i];
				    		i==0 ? showJson[itemName] = true : showJson[itemName] = false;
				    }
				    return showJson;
				},
				//递归运算
				getJson: function(item) {
					var json = {};
				    var length = Object.keys(item).length;
				    for(var j=0; j<length; j++) {
				        var name = Object.keys(item)[j];
				        var value = item[name];
			          	if(value.type !== 'object') {
						  	if(value.type == 'array') {
						  		var arr = [];
						  		if(!(value.items.type == 'object' || value.items.type == 'array')) {
						  			if(!_.isUndefined(value.items.default) && !_.isEmpty(value.items.default)) {
						  				arr.push(value.items.default);
						  				json[name] = arr;
						  			}
						  		} else {
						  			var arr_json = this.getJson(value.items.properties);
						  			arr.push(arr_json);
						  			json[name] = arr;
						  		}
						  	}else {
						  		if(value.type === 'number' && value.default && typeof value.default === 'string') {
						  			value.default = parseInt(value.default);
						  		}
						  		json[name] = value.default;
						  	}
						}
				        else {
				          	var item_json = this.getJson(value.properties);
				            json[name] = item_json;
				        }
				    }
				    return json;
				},

				changeJson: function(item) {
					var config={};
				    var length = Object.keys(item).length;
				    for(var i=0; i<length; i++) {
				        var name = Object.keys(item)[i];
				        var value = item[name];
				        var json = this.getJson(value.properties);
				        config[name] = json;
				    	}
				    return config;
				},

				getIsDisabled: function(item, required) {
					var length = Object.keys(item).length;
					for(var j=0; j<length; j++) {
				        var name = Object.keys(item)[j];
				        var value = item[name];
			          	if(value.type !== 'object') {
						  	if(value.type === 'array') {
						  		if(value.items.type === 'object' || value.items.type === 'array') {
						  			if(this.getIsDisabled(value.items.properties, value.items.required) === 'disabled') {
						  				return 'disabled';
						  			}
						  		}else {
						  			if(value.required && value.items.default) {
						  				return 'disabled';
						  			}
						  		}
						  	}else {
						  		var index;
						  		if(required) {
						  			index = required.indexOf(name);
						  		}

						  		if(index !== -1 && index >= 0) {
									if(value.default === undefined || value.default === null) {
						  				return 'disabled';
						  			}
						  		}
						  	}
						}
				        else {
				          	if(this.getIsDisabled(value.properties, value.required) === 'disabled') {
				          		return 'disabled'
				          	}
				        }
				    }
				},

				isDisabled: function(item) {
					var length = Object.keys(item).length;
					var result = '';
					for(var i=0; i<length; i++) {
				        var name = Object.keys(item)[i];
				        var value = item[name];
				        result = this.getIsDisabled(value.properties, value.required);
				    }
					return result;
				},

				prompt: function($scope) {
					$uibModal.open({
						templateUrl: 'templates/framework/prompt.html',
						controller: 'PromptController',
						backdrop: 'static',
						size: 'sm',
						resolve:{
							model: function() {
								return $scope.prompt;
							}
						}
					})
				}
	
	    	}
     }]);
});