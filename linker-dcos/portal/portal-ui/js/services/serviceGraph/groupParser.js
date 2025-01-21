define(['app'], function (app) {
    'use strict';
     app.provide.factory('componentTreeUtilService', ['$http','$q','treeUtilService',function ($http,$q,treeUtilService) {
			//group parser
			var parseGroup = function(rootPath,target,nodes,relations,extra_relations,zoomingRate){
				getNodesAndRelations(rootPath,target,nodes,relations);
				relations = clearRelations(relations);
				handleMultipleDependson(rootPath,relations,extra_relations);
				setNodeLevel(rootPath,nodes,relations,1);
				// setLevelHeight(nodes,treetype,levelHeights,zoomingRate);
				chainNodes(rootPath,nodes,relations);
				return relations;
			}
			
			var getNodesAndRelations = function(parentPath,target,nodes,relations){
				var parentnode = {
					"id": parentPath,
					"data": {
						"pathid": parentPath,
						"name": treeUtilService.idToSimple(parentPath),
						"imageSrc" : "img/graph/Appserver.png",
						"isGroup" : true
					},
					"children": []
				}	
				nodes.push(parentnode);
				
				_.each(target.apps, function(app) {
						var name = app.id;
						var pathid = treeUtilService.idToPath(name,parentPath);
						var deps = app.dependencies;
						
						var node = {
							"id": pathid,
							"data": {
								"pathid": pathid,
								"name": name,
								"imageSrc" : app.imageSrc,
								"instances" : app.instances,
								"isGroup" : false
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
							var dep_path_id = treeUtilService.idToPath(dep,parentPath);
							var rel = {
								"from" : pathid,
								"to" : dep_path_id,
								"type" : "depends"
							}
							relations.push(rel);
						});		
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

			var chainNodes = function(rootPath,nodes,relations){
				var rootnode = _.find(nodes,function(node){
					return node.id == rootPath;
				});
				chainSubNodes(rootPath,nodes,relations);
			}
			
			var chainSubNodes = function(parentPath,nodes,relations) {
				var roads = _.filter(relations,function(rel){
					return rel.from == parentPath;
				});
				var tonodes = _.map(roads,"to");
				var children = _.filter(nodes,function(node){
					return tonodes.indexOf(node.id) >= 0;		
				});
				
				_.each(children, function(child) {
					chainSubNodes(child.id,nodes,relations);
				});
				
				var fromnode = _.find(nodes,function(node){
					return node.id == parentPath;
				});
				fromnode.children = children;
			}
			//group parser end
			
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
								"y" : fromnode.pos.y + fromnode.Node.height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y - tonode.Node.height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.Node.height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y - tonode.Node.height/4
							}
						}else{
							direct = "up";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.Node.height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y + tonode.Node.height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.Node.height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y + tonode.Node.height/4
							}
						}
					}
					if(fromnode.pos.x < tonode.pos.x){
						if(fromnode.pos.y < tonode.pos.y){
							direct = "right";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.Node.height/4
							}
							to = {
								"x" : tonode.pos.x - 77 * scope.zoomingRate,
								"y" : tonode.pos.y - tonode.Node.height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.Node.height/4
							}
							turn2 = {
								"x" : tonode.pos.x - 117 * scope.zoomingRate,	
								"y" : tonode.pos.y - tonode.Node.height/4
							}
						}else{
							direct = "right";
							from = {
								"x" : fromnode.pos.x + 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.Node.height/4
							}
							to = {
								"x" : tonode.pos.x - 77 * scope.zoomingRate,
								"y" : tonode.pos.y + tonode.Node.height/4
							}
							turn1 = {
								"x" : fromnode.pos.x + 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.Node.height/4
							}
							turn2 = {
								"x" : tonode.pos.x - 117 * scope.zoomingRate,	
								"y" : tonode.pos.y + tonode.Node.height/4
							}
						}
					}
					if(fromnode.pos.x > tonode.pos.x){
						if(fromnode.pos.y < tonode.pos.y){
							direct = "left";
							from = {
								"x" : fromnode.pos.x - 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.Node.height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y - tonode.Node.height/4
							}
							turn1 = {
								"x" : fromnode.pos.x - 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y + fromnode.Node.height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y - tonode.Node.height/4
							}
						}else{
							direct = "left";
							from = {
								"x" : fromnode.pos.x - 77 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.Node.height/4
							}
							to = {
								"x" : tonode.pos.x + 77 * scope.zoomingRate,
								"y" : tonode.pos.y + tonode.Node.height/4
							}
							turn1 = {
								"x" : fromnode.pos.x - 117 * scope.zoomingRate,	
								"y" : fromnode.pos.y - fromnode.Node.height/4
							}
							turn2 = {
								"x" : tonode.pos.x + 117 * scope.zoomingRate,	
								"y" : tonode.pos.y + tonode.Node.height/4
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
			
			return {
				'parseGroup' : parseGroup,
				'drawReuseLine' : drawReuseLine
			}	
     }]);
});