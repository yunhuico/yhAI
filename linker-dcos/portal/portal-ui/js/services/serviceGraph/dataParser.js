define(['app'], function (app) {
    'use strict';
     app.provide.factory('treeUtilService', ['$http','$q',function ($http,$q) {
	    	var idToSimple = function(id) {
				return id.substring(id.lastIndexOf("/") + 1);
			};
			
			var idToPath = function(id,pid){
				return pid + "/" + id;
			}
			
			var depToPath = function(depid,pathid){
				var paths = pathid.split("/");
				var deppaths = depid.split("/");
				var backnum = _.filter(deppaths,function(path){
					return path == "..";
				}).length;
				var prefixnum = paths.length - backnum;
				var suffixnum = deppaths.length - backnum;
				var depPathID = _.first(paths,prefixnum).join("/") + "/" + _.last(deppaths,suffixnum).join("/");
				return depPathID;
			}
			
			var pathToDep = function(frompath,topath){
				var from = frompath.split("/");
				var to = topath.split("/");
				var prefix="";
				var result;
				for(var i=1;i>=1;i++){
					var testfrom = _.first(from,from.length-i);
					var testfrompath = testfrom.join("/");
					prefix += "../";
					if(topath.indexOf(testfrompath)==0){
						var tosuffix = topath.substring(testfrompath.length);
						result = prefix.substring(0,prefix.length-1) + tosuffix;
						break;
					}
				}
				return result;
			}
			
			var idToSGID = function(id){
				var paths = id.split("/");
				var sgid = paths[0] + "/" + paths[1]; 
				return sgid;
			}
			
			var idPrefix = function(id) {
				return id.substring(0,id.indexOf("-") );
			}

			var idSuffix = function(id) {
				return id.substring(id.indexOf("-")+1 );
			}

			var allocateImageToApp = function(group){
				if(_.isUndefined(group.groups) || group.groups == null || !_.isUndefined(group.apps)){
					_.each(group.apps,function(app){
					    if(app.id == "nginx"){
					    	app.imageSrc = "img/graph/app/nginx.png";
					    }else if(app.id == "zookeeper"){
					    	app.imageSrc = "img/graph/app/zookeeper.png";
					    }else if(app.id == "haproxy"){
					    	app.imageSrc = "img/graph/app/haproxy.png";
					    }else if(app.id.indexOf("mysql")>=0){
					    	app.imageSrc = "img/graph/app/mysql.png";
					   	}else{
					    	app.imageSrc = "img/graph/app/default.png";
					    }
					});
				}else{
					_.each(group.groups,function(subgroup){
						allocateImageToApp(subgroup);
					});
				}	
			}
			
			//service group parser
			var parseService = function(rootPath,target,nodes,relations,extra_relations,levelHeights,zoomingRate,treetype){
				getNodesAndRelations(rootPath,target,nodes,relations);
				relations = clearRelations(relations);
				handleMultipleDependson(rootPath,relations,extra_relations);
				setNodeLevel(rootPath,nodes,relations,1);
				setLevelHeight(nodes,treetype,levelHeights,zoomingRate);
				calNodePosition(rootPath,nodes,relations,levelHeights,treetype,zoomingRate);
				return relations;
			}
			
			var getNodesAndRelations = function(parentPath,target,nodes,relations){
				var parentnode = {
					"id": parentPath,
					"data": {
						"pathid": parentPath,
						"name": idToSimple(parentPath),
						"isTemplate" : true,
						"imageSrc" : "img/graph/Appserver.png",
						"apps": []
					},
					"children": []
				}	
				nodes.push(parentnode);
				
				_.each(target.groups, function(group) {
					if(_.isUndefined(group.groups) || group.groups == null || !_.isUndefined(group.apps)){
						var name = group.id;
						var pathid = idToPath(name,parentPath);
						var deps = group.dependencies;
						var apps = _.isUndefined(group.apps) ? [] : group.apps;
						
						var node = {
							"id": pathid,
							"data": {
								"pathid": pathid,
								"name": name,
								"isTemplate" : false,
								"apps": apps
							},
							"children": []
						}
						nodes.push(node);
						
						var relation = {
							"from" : parentPath,
							"to" : pathid,
							"type" : "contains"
						}
						relations.push(relation);
						
						_.each(deps, function(dep) {
							var dep_path_id = depToPath(dep,pathid);
							var rel = {
								"from" : pathid,
								"to" : dep_path_id,
								"type" : "depends"
							}
							relations.push(rel);
						});
					}else{
						var name = group.id;
						var pathid = idToPath(name,parentPath);
						var relation = {
							"from" : parentPath,
							"to" : pathid,
							"type" : "contains"
						}
						relations.push(relation);
						getNodesAndRelations(pathid,group,nodes,relations);
					}	
				});
			}
			
			var clearRelations = function(relations){
				var finalRelations = [];
				_.each(relations,function(relation){
					if(relation.type == "contains"){
						var tonode = relation.to;
						var alldependson = _.filter(relations,function(rel){
							return rel.type == "depends";
						});
						if(_.map(alldependson,"to").indexOf(tonode)<0){
							finalRelations.push(relation);
						}
					}else{
						finalRelations.push(relation);
					}
				});
				return finalRelations;
			}
			
			var handleMultipleDependson = function(rootPath,relations,extra_relations){
				var handledMultipleToNodes = [];
				_.each(relations,function(relation){
					var tonode = relation.to;
					if(relation.type == "depends" && handledMultipleToNodes.indexOf(tonode)<0){
						var allDependsOnTonode = _.where(relations,{"to":tonode,"type":"depends"});
						if(allDependsOnTonode.length>1){
							_.each(allDependsOnTonode,function(rel){
								var level = 1;
								level = getNodeLevel([rootPath],rel.from,relations,level);
								rel.level = level;
							});
							var newMultipleRels = shuffleMultipleRel(allDependsOnTonode);
							handledMultipleToNodes.push(newMultipleRels[0].to);
							newMultipleRels = _.last(newMultipleRels,newMultipleRels.length-1);
							
							_.each(newMultipleRels,function(item){
								extra_relations.push(item);
								var otherRel = _.find(relations,function(r){
									return r.from == item.from && r.to == item.to;
								});
								otherRel.needToRemove = true;
							})
						}
					}
				});
				
				for(var i = relations.length -1; i >= 0 ; i--){
				    if(relations[i].needToRemove){
				        relations.splice(i,1);
				    }
				}
			}
			
			var getNodeLevel = function(rootpaths,targetpath,relations,level){
				var roads = _.filter(relations,function(relation){
					return rootpaths.indexOf(relation.from) >= 0;
				});
				var tonodes = _.map(roads,"to");
				if(tonodes.indexOf(targetpath)<0){
					return getNodeLevel(tonodes,targetpath,relations,level+1);
				}else{
					return level;
				}
			}
			
			var shuffleMultipleRel = function(multipleRels){
				for(var i = multipleRels.length; i >0; i--){
					var targetRels = _.first(multipleRels,i);
					var index;
					if(i%2==1){
						index = (i-1)/2;
					}else{
						index = i/2 -1;
					}
					var middleOne = targetRels[index];
					multipleRels.splice(index,1);
					multipleRels.push(middleOne);
				}
				var sortedRels = _.sortBy(multipleRels,"level");
				return sortedRels;
			}
			
			var setNodeLevel = function(frompath,nodes,relations,level){
				var fromnode = _.find(nodes,function(node){
					return node.id == frompath;
				});
				fromnode.data.level = level;
				var roads = _.filter(relations,function(relation){
					return relation.from == frompath;
				});
				var tonodes = _.map(roads,"to");
				_.each(tonodes,function(tonode){
					setNodeLevel(tonode,nodes,relations,level+1);
				});
			}
			
			var setLevelHeight = function(nodes,treetype,levelHeights,zoomingRate){
				for(var i=1;i>=1;i++){
					var levelNodes = _.filter(nodes,function(node){
						return node.data.level == i;
					});
					if(levelNodes.length == 0){
						break;
					}
					_.each(levelNodes, function(node) {
						var appslen;
						if (treetype == "model") {
							appslen = node.data.apps.length > 0 ? node.data.apps.length : 1;
						} else {
							appslen = 0;
							_.each(node.data.apps, function(app) {
								appslen = appslen + app.instances;
							});
							appslen = appslen > 0 ? appslen : 1;
						}
				
						var h_node = appslen * getZoomingValue("node","height",zoomingRate);
						var index = _.map(levelHeights, "level").indexOf(i);
						if (index >= 0) {
							levelHeights[index].height = levelHeights[index].height + 100 * zoomingRate + h_node;
						} else {
							var _height = {
								"level": i,
								"height": h_node,
								"drawnFrom": 0
							}
							levelHeights.push(_height);
						}
					});
				}
			}
			
			var calNodePosition = function(rootPath,nodes,relations,levelHeights,treetype,zoomingRate){
				var y = 0;
				var rootnode = _.find(nodes,function(node){
					return node.id == rootPath;
				});
				rootnode.data.y = y;
				calChildPosition(rootPath,nodes,relations,levelHeights,treetype,zoomingRate,y);
			}
			
			var calChildPosition = function(parentPath,nodes,relations,levelHeights,treetype,zoomingRate,base_y) {
				var roads = _.filter(relations,function(rel){
					return rel.from == parentPath;
				});
				var tonodes = _.map(roads,"to");
				var children = _.filter(nodes,function(node){
					return tonodes.indexOf(node.id) >= 0;		
				});
				
				_.each(children, function(child) {
					var level = child.data.level;
					var nodeH;
					var heightPerNode = getZoomingValue("node","height",zoomingRate);
					if (treetype == "model") {
						nodeH = child.data.apps.length * heightPerNode > 0 ? child.data.apps.length * heightPerNode : heightPerNode;
					} else {
						var appslen = 0;
						_.each(child.data.apps, function(app) {
							appslen = appslen + app.instances;
						});
						appslen = appslen > 0 ? appslen : 1;
						nodeH = appslen * heightPerNode;
					}
					var levelH = _.find(levelHeights, function(lh) {
						return lh.level == level;
					});
					var wholeHeight = levelH.height;
					var drawnFrom = levelH.drawnFrom;
					var y = base_y - (wholeHeight / 2) + drawnFrom + (nodeH / 2);
					levelH.drawnFrom += nodeH + 100 * zoomingRate;
					child.data.y = y;
					calChildPosition(child.id,nodes,relations,levelHeights,treetype,zoomingRate,base_y);
				});
				
				var fromnode = _.find(nodes,function(node){
					return node.id == parentPath;
				});
				fromnode.children = children;
			}
			//service group parser end
			
			//draw reuse arrow
			var drawReuseLine = function(scope){
				var self = this;
				_.each(scope.extra_relations,function(line){
					var fromid = line.from;
					var toid = line.to;
					var fromnode = scope.st.graph.getNode(fromid);
					var tonode = scope.st.graph.getNode(toid);
					
					var canvas = scope.st.canvas;
					var direct,from,to,turn1,turn2;
					if(fromnode.pos.x == tonode.pos.x){
						if(fromnode.pos.y < tonode.pos.y){
							direct = "down";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.data.$height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y - tonode.data.$height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.data.$height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y - tonode.data.$height/4
							}
						}else{
							direct = "up";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.data.$height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y + tonode.data.$height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.data.$height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y + tonode.data.$height/4
							}
						}
					}
					if(fromnode.pos.x < tonode.pos.x){
						if(fromnode.pos.y < tonode.pos.y){
							direct = "right";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.data.$height/4
							}
							to = {
								"x" : tonode.pos.x - 77 * scope.zoomingRate,
								"y" : tonode.pos.y - tonode.data.$height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.data.$height/4
							}
							turn2 = {
								"x" : tonode.pos.x - 117 * scope.zoomingRate,	
								"y" : tonode.pos.y - tonode.data.$height/4
							}
						}else{
							direct = "right";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.data.$height/4
							}
							to = {
								"x" : tonode.pos.x - 77 * scope.zoomingRate,
								"y" : tonode.pos.y + tonode.data.$height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.data.$height/4
							}
							turn2 = {
								"x" : tonode.pos.x - 117 * scope.zoomingRate,	
								"y" : tonode.pos.y + tonode.data.$height/4
							}
						}
					}
					if(fromnode.pos.x > tonode.pos.x){
						if(fromnode.pos.y < tonode.pos.y){
							direct = "left";
							from = {
								"x" : fromnode.pos.x - 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.data.$height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y - tonode.data.$height/4
							}
							turn1 = {
								"x" : fromnode.pos.x - 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.data.$height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y - tonode.data.$height/4
							}
						}else{
							direct = "left";
							from = {
								"x" : fromnode.pos.x - 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.data.$height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y + tonode.data.$height/4
							}
							turn1 = {
								"x" : fromnode.pos.x - 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.data.$height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y + tonode.data.$height/4
							}
						}
					}
					drawArrow(from,to,turn1,turn2,canvas,direct);
				})
			}
			
			var drawArrow = function(from, to, turn1, turn2, canvas, direct){
		        var ctx = canvas.getCtx();

		        var v1,v2;
		        if(direct == "right"){
		        		v1 = {
		        			x : to.x-7,
		        			y : to.y-3.5
		        		}
		        		v2 = {
		        			x : to.x-7,
		        			y : to.y+3.5
		        		}
		        }else{	
		        		v1 = {
		        			x : to.x+7,
		        			y : to.y-3.5
		        		}
		        		v2 = {
		        			x : to.x+7,
		        			y : to.y+3.5
		        		}
		        }
		        
		        ctx.strokeStyle="#787878";
		        ctx.fillStyle="#787878";
		        ctx.beginPath();
		        ctx.moveTo(from.x, from.y);
		        ctx.lineTo(turn1.x, turn1.y);
		        ctx.lineTo(turn2.x, turn2.y);
		        ctx.lineTo(to.x, to.y);
		        ctx.stroke();
		        ctx.beginPath();
		        ctx.moveTo(v1.x, v1.y);
		        ctx.lineTo(v2.x, v2.y);
		        ctx.lineTo(to.x, to.y);
		        ctx.closePath();
		        ctx.fill();
			}
			
			//zooming util
			var zoomingObj = {
				"node" : [
					{
						"name" : "width",
						"value" : 153
					},
					{
						"name" : "height",
						"value" : 104
					}
				],
				"app_image_container" : [
					{
						"name" : "width",
						"value" : 126
					},
					{
						"name" : "height",
						"value" : 53
					}
				],
				"service_group_image_container" : [
					{
						"name" : "width",
						"value" : 146
					},
					{
						"name" : "height",
						"value" : 73
					}
				],
				"app_item_label": [
					{
						"name" : "font-size",
						"value" : 14
					},
					{
						"name" : "margin-top",
						"value" : 6
					},
				],
				"superscript" : [
					{
						"name" : "margin-bottom",
						"value" : 2
					},
					{
						"name": "margin-right",
						"value": 5
					},
				],
				"group_label" : [
					{
						"name" : "font-size",
						"value" : 16
					},
					{
						"name": "margin-top",
						"value": 10
					},
				],
				"group" : [
					{
						"name" : "min-height",
						"value" : 104
					},
					{
						"name" : "width",
						"value" : 153
					},
				],
				"group_app_item" : [
					{
						"name" : "margin",
						"value" : 10
					},
					{
						"name" : "width",
						"value" : 133
					},
					{
						"name" : "height",
						"value" : 84
					},
					{
						"name" : "padding-left",
						"value" : 5
					}
				]
			}
			
			var getZoomingCssString = function(target,zoomingRate){
				var targetObj = zoomingObj[target];
				var cssstring = "";
				_.each(targetObj,function(item){
					cssstring += item.name + ":" + item.value * zoomingRate + "px;"
				});
				return cssstring;
			}
			
			var getZoomingValue = function(target,propertyname,zoomingRate){
				var targetObj = zoomingObj[target];
				var property = _.find(targetObj,function(item){
					return item.name == propertyname;
				});
				return property.value * zoomingRate;
			}
			//zooming util end
			
			//service group data util
			var findGroupByPath = function(scope,groupPathID,fromInstance){
				var self = this;
				var group_paths = groupPathID.split("/");
				var target_group = scope.selectedModel;
				if(fromInstance == true){
					target_group = scope.selectedService;
				}
				for(var i=2;i<group_paths.length;i++){
					var groupid = group_paths[i];
					target_group = findGroupByID(target_group,groupid);
				}
				return target_group;
			}
			
			var findGroupByID = function(parentgroup,groupid){
				return _.find(parentgroup.groups,function(group){
					return group.id == groupid;
				});
			}
			
			var isDepGroupIDDuplicated = function(scope,p_gid,p_gtype,gid){
				var self = this;
				var parentgroup = findParentGroup(scope,p_gid,p_gtype,true);
				return _.map(parentgroup.groups, "id").indexOf(gid) >= 0 ;
			}
			
			var isNewGroupIDDuplicated = function(scope,p_gid,p_gtype,gid){
				var self = this;
				if(idToSimple(p_gid) == gid){
					return false;
				}
				var parentgroup = findParentGroup(scope,p_gid,p_gtype,false);
				return _.map(parentgroup.groups, "id").indexOf(gid) >= 0 ;
			}
			
			var findParentGroup = function(scope,p_gid,p_gtype,addDep){
				var self = this;
				var parentgroup;
				if(p_gid.split("/").length == 2){
					parentgroup = scope.selectedModel;
				}else if(p_gtype == "normalgroup" || !addDep){
					p_gid = p_gid.substring(0,p_gid.lastIndexOf("/"));
					parentgroup = findGroupByPath(scope,p_gid);
				}else if(addDep){
					parentgroup = findGroupByPath(scope,p_gid);
				}
				return parentgroup;
			}
			
			var notAllowBillingForSubTemplate = function(group){
				_.each(group.groups,function(_group){
					if(!_.isUndefined(_group.groups) && _group.groups != null){
					   _group.billing = false;
					   notAllowBillingForSubTemplate(_group);
					}
				});
			}
			
			var groupIDIsValid = function(gid){
				var regExp = /^(([a-z0-9]|[a-z0-9][a-z0-9\\-]*[a-z0-9])\\.)*([a-z0-9]|[a-z0-9][a-z0-9\\-]*[a-z0-9])$/;
				return regExp.test(gid);
			}
			//service group data util

			var findAppInSelectedModel = function(scope,gid, aid,fromInstance) {
				var _group = findGroupByPath(scope,gid,fromInstance);
				var _app = _.find(_group.apps, function(app) {
					return app.id == aid;
				});
				return _app;
			}
			
			var isJson = function(str) {
				try {
					JSON.parse(str);
				} catch (e) {
					return false;
				}
				return true;
			}
			
			var parsePrefix = function(){
		        var currentUserName = localStorage.username;
		        var parsedPrefix = "";
		        if(currentUserName != "sysadmin"){
		        	parsedPrefix = currentUserName.replace(/@/g, "_at_");
		        	parsedPrefix = parsedPrefix.replace(/\./g, "_");
		        }
		        return parsedPrefix;
			}
			
			var isReverseDirection = function(scope,from,to){
				var tonode = findGroupByPath(scope,to);
				var depid = pathToDep(to,from);
				if(!_.isUndefined(tonode.dependencies) && tonode.dependencies.indexOf(depid)>=0){
					return true;
				}else{
					return false;
				}
			}
			
			var isSameParent = function(id1,id2){
				return id1.substring(0,id1.lastIndexOf("/")) == id2.substring(0,id2.lastIndexOf("/"));
			}
			
			return {
				'idToSimple' : idToSimple,
				'idToPath' : idToPath,
				'depToPath' : depToPath,
				'pathToDep' : pathToDep,
				'idToSGID' : idToSGID,
				'idPrefix' : idPrefix,
				'idSuffix' : idSuffix,
				'allocateImageToApp' : allocateImageToApp,
				'parseService' : parseService,
				'drawReuseLine' : drawReuseLine,
				'getZoomingCssString' : getZoomingCssString,
				'getZoomingValue' : getZoomingValue,
				'findGroupByPath' : findGroupByPath,
				'isDepGroupIDDuplicated' : isDepGroupIDDuplicated,
				'isNewGroupIDDuplicated' : isNewGroupIDDuplicated,
				'findParentGroup' : findParentGroup,
				'notAllowBillingForSubTemplate' : notAllowBillingForSubTemplate,
				'groupIDIsValid' : groupIDIsValid,
				'findAppInSelectedModel' : findAppInSelectedModel,
				'isJson' : isJson,
				'parsePrefix' : parsePrefix,
				'isReverseDirection' : isReverseDirection,
				'isSameParent' : isSameParent
			}	
     }]);
});