define(['app','jit','directives/graphNode','services/serviceGraph/dataParser','services/serviceGraph/groupParser'], function (app) {
    'use strict';
     app.provide.factory('ServiceGraphService', ['$http','$q','$compile','treeUtilService','componentTreeUtilService',function ($http,$q,$compile,treeUtilService,componentTreeUtilService) {
	    	return {              
			    drawGraph : function(elementid,treejson,$scope) {
			        var self = this;

					//reset designer height
					var height = _.max(_.map($scope.levelHeights, "height"));
					if ($scope.modelDesignerHeight < height + 300) {
						$scope.modelDesignerHeight = height + 300;
					}
				
					var offsetx = $('#'+elementid).width() / 2 - 100,offsety = 0;
					if(treejson.children.length > 0){
				    		offsetx = $('#'+elementid).width() / 2 - 100;
					}
					
					self.setRealOffset($scope);
					
					$("#"+elementid).empty();
					$("#"+elementid).height($scope.modelDesignerHeight);
					//Create a new ST instance  
					var st = new $jit.ST({  
					    //id of viz container element  
					    siblingOffset : 50,
					    injectInto: elementid,  
					    //set duration for the animation  
					    constrained : false,
					    levelsToShow : 50,
					    offsetX : offsetx,
					    offsetY : offsety,
					    duration: 0,  
					    //set animation transition type  
					    transition: $jit.Trans.Quart.easeInOut,  
					    //set distance between node and its children  
					    levelDistance: 80 * $scope.zoomingRate,  
						
					    //enable panning  
					    Navigation: {  
					      enable:true,  
					      panning:true  
					    },  
					    //set node and edge styles  
					    //set overridable=true for styling individual  
					    //nodes or edges  
					    Node: {  
					        height: $scope.getZoomingValue("node","height"),  
					        width: $scope.getZoomingValue("node","width"),  
					        overridable: true  
					    },  
					      
					    Edge: {  
					        type: 'arrow',  
					  		lineWidth: 2, 
					  		color: '#787878',
					  		dim : '8',
					        overridable: true  
					    },  
					      
					    //This method is called on DOM label creation.  
					    //Use this method to add event handlers and styles to  
					    //your node.  
					    onCreateLabel: function(label, node){  
					        label.id = node.id;   
					        var istemplate = node.data.isTemplate;
					        if(istemplate){        	
					        		$scope.node = node;
					        		var html = $compile('<servicegroupnode></servicegroupnode>')($scope);
					        		$scope.$apply();

					        		label.innerHTML = html[0].innerHTML;
					        }else{
					      	  	$scope.node = node;
					        		var html = $compile('<normalgroupnode></normalgroupnode>')($scope);
					        		$scope.$apply();

					        		label.innerHTML = html[0].innerHTML;
					        }
					    },  
					    onPlaceLabel: function(label, node, controllers){          
				            //override label styles
				            var style = label.style;  
				            // show the label and let the canvas clip it
				            style.display = '';
				       },
					    onBeforePlotNode: function(node){  
					         node.data.$color = "#fff"; 
					         var heightPerApp = $scope.getZoomingValue("node","height");
					         node.data.$height = node.data.apps.length *  heightPerApp> 0 ? node.data.apps.length * heightPerApp : heightPerApp;
					         node.pos.y = node.data.y;
					    }, 
					    onBeforePlotLine: function(adj){  
					        adj.data.$color = "#787878";
					        treeUtilService.drawReuseLine($scope);
					    }
					});  
					//load json data  
					st.loadJSON(treejson);  
					//compute node positions and layout  
					st.compute();
					//optional: make a translation of the tree  
					st.geom.translate(new $jit.Complex(-200, 0), "current");  
					//emulate a click on the root node.  
					st.onClick(st.root,{
						 Move: {
				            enable: true,
				            offsetX: _.isUndefined($scope.st) ? offsetx : offsetx - $scope.canvasX,
				            offsetY: _.isUndefined($scope.st) ? offsety : offsety - $scope.canvasY
				        }
					});
					
					$scope.st = st;
					
					setTimeout(function(){
						treeUtilService.drawReuseLine($scope);
			   			self.bindEventsForNodes($scope);
					},500)
			    },

			   setRealOffset : function($scope){
					if(_.isUndefined($scope.canvasX)){
						$scope.canvasX = 0;
					}
					if(_.isUndefined($scope.canvasY)){
						$scope.canvasY = 0;
					}
					if(!_.isUndefined($scope.st)){
						$scope.canvasX += $scope.st.canvas.translateOffsetX;
						$scope.canvasY += $scope.st.canvas.translateOffsetY;
					}
				},

				bindEventsForNodes : function($scope){		  		
			  		//for normal group
			  		var groupLabels = $(".group").parent().find(".glyphicon-search");
			  		_.each(groupLabels,function(el){
			  			el.addEventListener(
				        		'click',
				             function(e) {
				             	$scope.showComponentDependencies(e);
				             }
				         );
			  		});
			  	},

			  	drawComponentGraph : function(elementid,treejson,$scope) {
			        var self = this;
				
					var offsetx = $('#'+elementid).width() / 2 - 100,offsety = 0;
					if(treejson.children.length > 0){
				    		offsetx = $('#'+elementid).width() / 2 - 100;
					}
					
					self.setRealOffset($scope);
					
					$("#"+elementid).empty();

					treejson.id = treeUtilService.idToSimple(treejson.id);

					//Create a new ST instance  
					var st = new $jit.ST({  
					    //id of viz container element  
					    siblingOffset : 50,
					    injectInto: elementid,  
					    //set duration for the animation  
					    constrained : false,
					    levelsToShow : 50,
					    offsetX : offsetx,
					    offsetY : offsety,
					    duration: 0,  
					    //set animation transition type  
					    transition: $jit.Trans.Quart.easeInOut,  
					    //set distance between node and its children  
					    levelDistance: 80 * $scope.zoomingRate,  
						
					    //enable panning  
					    Navigation: {  
					      enable:true,  
					      panning:true  
					    },  
					    //set node and edge styles  
					    //set overridable=true for styling individual  
					    //nodes or edges  
					    Node: {  
					        height: $scope.getZoomingValue("node","height"),  
					        width: $scope.getZoomingValue("node","width"),  
					        overridable: true  
					    },  
					      
					    Edge: {  
					        type: 'arrow',  
					  		lineWidth: 2, 
					  		color: '#787878',
					  		dim : '8',
					        overridable: true  
					    },  
					      
					    //This method is called on DOM label creation.  
					    //Use this method to add event handlers and styles to  
					    //your node.  
					    onCreateLabel: function(label, node){  
					        label.id = node.id;   
					        var isGroup = node.data.isGroup;
					        if(isGroup){        	
					        	$scope.node = node;
					        	var html = $compile('<servicegroupnode></servicegroupnode>')($scope);
					        	$scope.$apply();

					        	label.innerHTML = html[0].innerHTML;
					        }else{
					      	  	$scope.node = node;
					        	var html = $compile('<singleappnode></singleappnode>')($scope);
					        	$scope.$apply();

					        	label.innerHTML = html[0].innerHTML;
					        }
					    },  
					    onPlaceLabel: function(label, node, controllers){          
				            //override label styles
				            var style = label.style;  
				            // show the label and let the canvas clip it
				            style.display = '';
				       },
					    onBeforePlotNode: function(node){  
					         node.data.$color = "#fff"; 
					         // var heightPerApp = $scope.getZoomingValue("node","height");
					         // node.data.$height = node.data.apps.length *  heightPerApp> 0 ? node.data.apps.length * heightPerApp : heightPerApp;
					         // node.pos.y = node.data.y;
					    }, 
					    onBeforePlotLine: function(adj){  
					        adj.data.$color = "#787878";
					        componentTreeUtilService.drawReuseLine($scope);
					    }
					});  
					//load json data  
					st.loadJSON(treejson);  
					//compute node positions and layout  
					st.compute();
					//optional: make a translation of the tree  
					st.geom.translate(new $jit.Complex(-200, 0), "current");  
					//emulate a click on the root node.  
					st.onClick(st.root,{
						 Move: {
				            enable: true,
				            offsetX: _.isUndefined($scope.st) ? offsetx : offsetx - $scope.canvasX,
				            offsetY: _.isUndefined($scope.st) ? offsety : offsety - $scope.canvasY
				        }
					});
					
					$scope.st = st;
					
					setTimeout(function(){
						componentTreeUtilService.drawReuseLine($scope);
			   			// self.bindEventsForNodes($scope);
					},500)
			    }
	    	}
	    	
     }]);
});