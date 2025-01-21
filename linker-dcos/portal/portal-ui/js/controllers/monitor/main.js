define(['app','services/node/main','services/monitor/main'], function(app) {
    'use strict';
    app.controllerProvider.register('MonitorController', ['$scope', '$localStorage', '$uibModal', '$location', '$state', 'ResponseService', 'MonitorService', 'CommonService',
    	function($scope, $localStorage, $uibModal, $location, $state, ResponseService,MonitorService,CommonService) {

    	$scope.$storage = $localStorage;

    	$scope.getClusterNodes = function(){
    		console.log(CommonService.clusterAvailable());
    		if(CommonService.clusterAvailable()){
	    			if(CommonService.endPointAvailable()){
	    // 			var clusterendpoint = CommonService.getEndPoint();
					// var nodeip = clusterendpoint.substring(0,clusterendpoint.indexOf(":"));
					var nodeip = CommonService.getEndPoint();
	    			MonitorService.getNodes(nodeip).then(function(data) {			
						$scope.nodes = data.slaves;
						if($scope.nodes.length>0 && _.isUndefined($scope.$storage.selectedContainerToMonitor)){
							$scope.filter = {
								"type" : "node",
								"value" : $scope.nodes[0].hostname
							};

							if(!_.isUndefined($scope.$storage.selectedNodeToMonitor)){
								$scope.filter.value = $scope.$storage.selectedNodeToMonitor;
								delete $scope.$storage.selectedNodeToMonitor;
							}
							
							$scope.changeNode();
						}	
					}, function(errorMessage) {
						$scope.nodes = [];
					});
	    		}else{
	    			$scope.nodes = [];
	    		}
    		}else{
	    		$scope.nocluster = true;
	    	}	
    	}

    	$scope.getClusterServices = function(){
    		if (CommonService.endPointAvailable()) {
    			MonitorService.getServices(CommonService.getEndPoint()).then(function(data) {		
					$scope.services = data.data;
					if(!_.isUndefined($scope.$storage.selectedServiceToMonitor)){
						$scope.filter = {
							"type" : "service",
							"value" : $scope.$storage.selectedServiceToMonitor
						};

						delete $scope.$storage.selectedServiceToMonitor;
						$scope.selectedServiceContainer = {
							"value" : ""
						}
						$scope.changeService();
					}
				}, function(errorMessage) {
					$scope.services = [];
				});
    		}else{
    			$scope.services = [];
    		}	
    	}

    	$scope.changeType = function(){
			if($scope.filter.type == "node"){
				$scope.filter = {
					"type" : "node",
					"value" : $scope.nodes[0].hostname
				};
				$scope.changeNode();
			}
			else if($scope.filter.type == "service"){
				$scope.filter = {
					"type" : "service",
					"value" : $scope.services[0].name
				};
				$scope.selectedServiceContainer = {
					"value" : ""
				}
				$scope.changeService();
			}
    	}

		$scope.changeNode = function(){
			clearLocalSelectedValue();

			if($location.path() == "/monitor"){
				 if($scope.filter.type == "node"){
				 	$scope.getNodeSpec();
				 }else{
					$scope.getContainerSpec();
				 }
			}
		}

		function clearLocalSelectedValue(){
			if(!_.isUndefined($scope.selectednetwork) && !_.isUndefined($scope.selectednetwork.value)){
				$scope.selectednetwork.value = "";
			}
			if(!_.isUndefined($scope.selecteddisk) && !_.isUndefined($scope.selecteddisk.value)){
				$scope.selecteddisk.value = "";
			}
			if(!_.isUndefined($scope.selectedfs) && !_.isUndefined($scope.selectedfs.value)){
				$scope.selectedfs.value = "";
			}
		}

		$scope.changeService = function(){
			if (CommonService.endPointAvailable()) {
				var groupid = $scope.filter.value;
				MonitorService.getServiceContainers(CommonService.getEndPoint(),groupid).then(function(data) {
	    				_.each(data.data,function(item){
	    					item.label = item.appId.substring(item.appId.lastIndexOf("/")+1) + " (" + item.name.substring(item.name.indexOf(".")+1) + ")"
	    				})		
						$scope.servicecontainers = data.data;
						if($scope.servicecontainers.length>0){
							if(!_.isUndefined($scope.$storage.selectedContainerToMonitor)){
								$scope.selectedServiceContainer.value = $scope.$storage.selectedContainerToMonitor;
								delete $scope.$storage.selectedContainerToMonitor;
							}else{
								$scope.selectedServiceContainer.value = $scope.servicecontainers[0].name;
							}
							$scope.changeNode();	
						}else{
							$scope.hasMonitorData = false;
						}		
				}, function(errorMessage) {
					$scope.servicecontainers = [];
				});
			}else{
				$scope.servicecontainers = [];
			}	
		}

		$scope.getMachineInfo = function(ip,slaveid){
			MonitorService.machineInfo(ip,slaveid).then(function(data){
				$scope.machineInfo = data;
				console.log("$scope.filter.type ----");
				console.log($scope.filter.type);
				if($scope.filter.type == "node"){
					console.log('start run node monitor ---')
					$scope.nodeMonitor();
				}else{
					$scope.containerMonitor();
				}
				$scope.hasMonitorData = true;
			},
			function(errorMessage){
				clearTimeout(window.reloadClusterMonitorAction);
				$scope.hasMonitorData = false;
			});
		}

		$scope.getNodeSpec = function(){
			var target = _.filter($scope.nodes,function(node){
				return node.hostname == $scope.filter.value;
			});
			_.each(target,function(item){
				// var clusterendpoint = CommonService.getEndPoint();
				// var nodeip = clusterendpoint.substring(0,clusterendpoint.indexOf(":"));
				var nodeip = CommonService.getEndPoint();
				var slaveid = item.id;

				MonitorService.nodeSpec(nodeip,slaveid).then(function(data){
					$scope.nodeInfos = {
						"spec" : {},
						"data" : []
					};
					for (var key in data) {
				 		$scope.nodeInfos.spec = data[key];
					}
					$scope.getMachineInfo(nodeip,slaveid);
					$scope.hasMonitorData = true;
				},
				function(errorMessage){
					clearTimeout(window.reloadClusterMonitorAction);
					$scope.hasMonitorData = false;
				});
			}) 
		}

		$scope.getContainerSpec = function(){
			var target;
			if($scope.filter.type == "container"){
				target = _.filter($scope.containers,function(container){
					return container.containername == $scope.filter.value;
				});
			}else if($scope.filter.type == "service"){
				target = _.filter($scope.servicecontainers,function(container){
					return container.name == $scope.selectedServiceContainer.value;
				});
			}
			 
			_.each(target,function(item){
				// var clusterendpoint = CommonService.getEndPoint();
				// var nodeip = clusterendpoint.substring(0,clusterendpoint.indexOf(":"));
				var nodeip = CommonService.getEndPoint();
				var slaveid = item.slaveId;
				var dockername = item.name;

				MonitorService.containerSpec(nodeip,slaveid,dockername).then(function(data){
					$scope.nodeInfos = {
						"spec" : {},
						"data" : []
					};
					for (var key in data) {
				 		$scope.nodeInfos.spec = data[key];
					}
					$scope.hasMonitorData = true;
					$scope.getMachineInfo(nodeip,slaveid);
				},
				function(errorMessage){
					clearTimeout(window.reloadClusterMonitorAction);
					$scope.hasMonitorData = false;
				});
			}) 
		}

		$scope.nodeMonitor = function(){
			clearTimeout(window.reloadClusterMonitorAction);

			if($location.path() == "/monitor"){
				console.log('run monitor path')
				var target = _.filter($scope.nodes,function(node){
					return node.hostname == $scope.filter.value;
				});

				console.log(target);
				_.each(target,function(item){
					// var clusterendpoint = CommonService.getEndPoint();
					// var nodeip = clusterendpoint.substring(0,clusterendpoint.indexOf(":"));
					var nodeip = CommonService.getEndPoint();
					var slaveid = item.id;

					MonitorService.nodeMonitor(nodeip,slaveid).then(function(data){
						console.log(data);
						if(_.isUndefined(data) || _.isEmpty(data)){
							console.log(1)
							clearTimeout(window.reloadClusterMonitorAction);
							$scope.hasMonitorData = false;
							return;
						}else{
							console.log(2)
							if($scope.nodeInfos.data.length == 60){
								$scope.nodeInfos.data.shift();
							}
							for (var key in data) {
								console.log(3, key)
								if($scope.nodeInfos.data.length > 0 && data[key][0].timestamp == $scope.nodeInfos.data[$scope.nodeInfos.data.length-1].timestamp){
									console.log(4.1)
									window.reloadClusterMonitorAction = setTimeout($scope.nodeMonitor,1000);
								}else{
									console.log(4.2)
									$scope.nodeInfos.data.push(data[key][0]);
									$scope.drawMonitor();
									window.reloadClusterMonitorAction = setTimeout($scope.nodeMonitor,1000);
								}		
							}
						}	
					},
					function(errorMessage){
						console.log(4)
						clearTimeout(window.reloadClusterMonitorAction);
						$scope.hasMonitorData = false;
				    });
				}) 
			}
		}

		$scope.containerMonitor = function(){
			clearTimeout(window.reloadClusterMonitorAction);
			if($location.path() == "/monitor"){
				var target;
				if($scope.filter.type == "container"){
					target = _.filter($scope.containers,function(container){
						return container.containername == $scope.filter.value;
					});
				}else if($scope.filter.type == "service"){
					target = _.filter($scope.servicecontainers,function(container){
						return container.name == $scope.selectedServiceContainer.value;
					});
				}
				
				_.each(target,function(item){
					// var clusterendpoint = CommonService.getEndPoint();
					// var nodeip = clusterendpoint.substring(0,clusterendpoint.indexOf(":"));
					var nodeip = CommonService.getEndPoint();
					var slaveid = item.slaveId;
					var dockername = item.name;

					MonitorService.containerMonitor(nodeip,slaveid,dockername).then(function(data){
						if(_.isUndefined(data) || _.isEmpty(data)){
							clearTimeout(window.reloadClusterMonitorAction);
							$scope.hasMonitorData = false;
							return;
						}else{
							if($scope.nodeInfos.data.length == 60){
								$scope.nodeInfos.data.shift();
							}
							for (var key in data) {
								if($scope.nodeInfos.data.length > 0 && data[key][0].timestamp == $scope.nodeInfos.data[$scope.nodeInfos.data.length-1].timestamp){
									window.reloadClusterMonitorAction = setTimeout($scope.containerMonitor,1000);
								}else{
									$scope.nodeInfos.data.push(data[key][0]);
									$scope.drawMonitor();
									window.reloadClusterMonitorAction = setTimeout($scope.containerMonitor,1000);
								}		
							}
						}
					},
					function(errorMessage){
						clearTimeout(window.reloadClusterMonitorAction);
						$scope.hasMonitorData = false;
				    });
				}) 
			}
		}

		$scope.drawMonitor = function(){
			var steps = [];
			// CPU.
			if ($scope.nodeInfos.data[0].has_cpu) {
				steps.push(function() {
					drawCpuTotalUsage("cpu-total-usage-chart");
				});
				$scope.hascpu = true;
			}else{
				$scope.hascpu = false;
			}
		
			// Memory.
			if ($scope.nodeInfos.data[0].has_memory) {
				steps.push(function() {
					drawMemoryUsage("memory-usage-chart");
				});
				$scope.hasmemory = true;
			}else{
				$scope.hasmemory = false;
			}
		
			// Network.
			if ($scope.nodeInfos.data[0].has_network) {
				steps.push(function() {
					prepareDrawNetwork("network-chart");
				});
				// $scope.hasnetwork = true;
			}else{
				$scope.hasnetwork = false;
			}
		
			// Filesystem.
			if ($scope.nodeInfos.data[0].has_filesystem) {
				steps.push(function() {
		            prepareDrawFileSystemUsage("filesystem-usage-chart");
		        });
		        // $scope.hasfilesystem = true;
			}else{
				$scope.hasfilesystem = false;
			}

			// disk io.
			if ($scope.nodeInfos.data[0].has_diskio) {
				steps.push(function() {
		            prepareDrawDiskIO("diskio-chart");
		        });
		        // $scope.hasdiskio = true;
			}else{
				$scope.hasdiskio = false;
			}

			stepExecute(steps);
		}
		
		// Expects an array of closures to call. After each execution the JS runtime is given control back before continuing.
		// This function returns asynchronously
		function stepExecute(steps) {
			// No steps, stop.
			if (steps.length == 0) {
				return;
			}
		
			// Get a step and execute it.
			var step = steps.shift();
			step();
		
			// Schedule the next step.
			setTimeout(function() {
				stepExecute(steps);
			}, 0);
		}
		
		// Draw a line chart.
		function drawLineChart(seriesTitles, data, elementId, unit) {
			var min = Infinity;
			var max = -Infinity;
			for (var i = 0; i < data.length; i++) {
				// Convert the first column to a Date.
				if (data[i] != null) {
					data[i][0] = new Date(data[i][0]);
				}
		
				// Find min, max.
				for (var j = 1; j < data[i].length; j++) {
					var val = data[i][j];
					if (val < min) {
						min = val;
					}
					if (val > max) {
						max = val;
					}
				}
			}
		
			// We don't want to show any values less than 0 so cap the min value at that.
			// At the same time, show 10% of the graph below the min value if we can.
			var minWindow = min - (max - min) / 10;
			if (minWindow < 0) {
				minWindow = 0;
			}
		
			// Add the definition of each column and the necessary data.
			var dataTable = new google.visualization.DataTable();
			dataTable.addColumn('datetime', seriesTitles[0]);
			for (var i = 1; i < seriesTitles.length; i++) {
				dataTable.addColumn('number', seriesTitles[i]);
			}
			dataTable.addRows(data);
		
			// Create and draw the visualization.
			if (!(elementId in window.charts)) {
				window.charts[elementId] = new google.visualization.LineChart(document.getElementById(elementId));
			}
		
			var opts = {
				curveType: 'function',
				height: 300,
				focusTarget: "category",
				vAxis: {
					title: unit,
					viewWindow: {
						min: minWindow
					}
				},
				legend: {
					position: 'bottom'
				}
			};
			// If the whole data series has the same value, try to center it in the chart.
			if ( min == max) {
				opts.vAxis.viewWindow.max = 1.1 * max
				opts.vAxis.viewWindow.min = 0.9 * max
			}
		
			window.charts[elementId].draw(dataTable, opts);
		}
		
		// Gets the length of the interval in nanoseconds.
		function getInterval(current, previous) {
			var cur = new Date(current);
			var prev = new Date(previous);
		
			// ms -> ns.
			return (cur.getTime() - prev.getTime()) * 1000000;
		}

		// Checks if the specified stats include the specified resource.
		function hasResource(stats, resource) {
			return stats.length > 0 && stats[0][resource];
		}
		
		// Following the IEC naming convention
		function humanizeIEC(num) {
		    var ret = humanize(num, 1024, ["TiB", "GiB", "MiB", "KiB", "B"]);
			return ret[0].toFixed(2) + " " + ret[1];
		}
		
		// Following the Metric naming convention
		function humanizeMetric(num) {
		    var ret = humanize(num, 1000, ["TB", "GB", "MB", "KB", "Bytes"]);
			return ret[0].toFixed(2) + " " + ret[1];
		}

		function humanize(num, size, units) {
		    var unit;
		    for (unit = units.pop(); units.length && num >= size; unit = units.pop()) {
		        num /= size;
		    }
		    return [num, unit];
		}

		// Draw the graph for CPU usage.
		function drawCpuTotalUsage(elementId) {
			var titles = ["Time","Total Usage(总使用量)"];
			var data = []; 

			console.log("$scope.nodeInfos ---");
			console.log($scope.nodeInfos);
			if (hasResource($scope.nodeInfos.data, "cpu")) {
				for (var i = 1; i < $scope.nodeInfos.data.length; i++) {
					var cur = $scope.nodeInfos.data[i];
					var prev = $scope.nodeInfos.data[i - 1];
					var intervalInNs = getInterval(cur.timestamp, prev.timestamp);

					var rawUsage = cur.cpu.usage.total - prev.cpu.usage.total;
					var num_cores = $scope.machineInfo.num_cores;
				    // Convert to millicores and take the percentage
				    var cpuUsage = 0;
				    cpuUsage =
				        Math.floor(((rawUsage / intervalInNs) / num_cores) * 10000) / 100;
				    if (cpuUsage > 100) {
				      cpuUsage = 100;
				    }
						
					if(data.length < 59){
						var elements = [];
						elements.push(cur.timestamp);
						elements.push(cpuUsage);
						data.push(elements);
					}else{
						data[i-1].push(cpuUsage);
					}
				}

				console.log(data);
			}
			
			drawLineChart(titles, data, elementId, "%(百分比)");
		}
		
		function drawMemoryUsage(elementId) {
			var titles = ["Time","Total Usage(总使用量)"];
			var data = [];

			if (hasResource($scope.nodeInfos.data, "memory")) {
				for (var i = 0; i < $scope.nodeInfos.data.length; i++) {
					var cur = $scope.nodeInfos.data[i];
					var memory_limit = $scope.nodeInfos.spec.memory.limit;
					var memory_capacity = $scope.machineInfo.memory_capacity;
					var limit = Math.min(memory_limit,memory_capacity);
					var totalMemory = Math.floor((cur.memory.usage * 10000.0) / limit) / 100.0;
				
					if(data.length < 60){
						var elements = [];
						elements.push(cur.timestamp);
						elements.push(totalMemory);
						data.push(elements);
					}else{
						data[i].push(totalMemory);
					}
				}
			}
		
			drawLineChart(titles, data, elementId, "%(百分比)");
		}
		
		//network
		function prepareDrawNetwork(elementId){
			$scope.networks = [];
			if(_.isUndefined($scope.selectednetwork)){
				$scope.selectednetwork = {
					value : ""
				};
			}
			
			var currentnetworks = $scope.nodeInfos.data[$scope.nodeInfos.data.length-1].network.interfaces;
			_.each($scope.machineInfo.network_devices,function(nw,index){
				if(_.map(currentnetworks,"name").indexOf(nw.name) >= 0){
					$scope.networks.push(nw);
					if($scope.networks.length == 1 && $scope.selectednetwork.value == ""){
						$scope.selectednetwork.value = nw.name;
					}
				}	
			});
			if($scope.networks.length > 0){
				$scope.hasnetwork = true;
				drawNetwork(elementId);
			}else{
				$scope.hasnetwork = false;
			}	
		}

		$scope.changeNetwork = function(){
			drawNetwork("network-chart");
		}

		function drawNetwork(elementId) {
			var titles = ["Time","Tx bytes(发送字节)","Rx bytes(接收字节)"];
			var data = [];
			
			if (hasResource($scope.nodeInfos.data, "network")) {
				for (var i = 1; i < $scope.nodeInfos.data.length; i++) {
					var cur = $scope.nodeInfos.data[i];
					var prev = $scope.nodeInfos.data[i - 1];
					var intervalInSec = getInterval(cur.timestamp, prev.timestamp) / 1000000000;
					var c_index = _.map(cur.network.interfaces,"name").indexOf($scope.selectednetwork.value);
					var p_index = _.map(prev.network.interfaces,"name").indexOf($scope.selectednetwork.value);

					if(data.length < 59){
						var elements = [];
						elements.push(cur.timestamp);
						elements.push((cur.network.interfaces[c_index].tx_bytes - prev.network.interfaces[p_index].tx_bytes) / intervalInSec);
						elements.push((cur.network.interfaces[c_index].rx_bytes - prev.network.interfaces[p_index].rx_bytes) / intervalInSec);
						data.push(elements);		
					}else{
						data[i-1].push((cur.network.interfaces[c_index].tx_bytes - prev.network.interfaces[p_index].tx_bytes) / intervalInSec);
						data[i-1].push((cur.network.interfaces[c_index].rx_bytes - prev.network.interfaces[p_index].rx_bytes) / intervalInSec);
					}
				}
			}
			
			drawLineChart(titles, data, elementId, "Bytes per second(每秒字节数)");
		}
		
		//filesystem	
		function prepareDrawFileSystemUsage(elementId){
			$scope.filesystems = [];
			if(_.isUndefined($scope.selectedfs)){
				$scope.selectedfs = {
					value : ""
				};
			}
			
			var currentfs = $scope.nodeInfos.data[$scope.nodeInfos.data.length-1].filesystem;
			_.each($scope.machineInfo.filesystems,function(fs,index){
				if(_.map(currentfs,"device").indexOf(fs.device) >= 0){
					$scope.filesystems.push(fs);
					if($scope.filesystems.length == 1 && $scope.selectedfs.value == ""){
						$scope.selectedfs.value = fs.device;
					}
				}
			});
			if($scope.filesystems.length > 0){
				$scope.hasfilesystem = true;
				drawFileSystemUsage(elementId);
			}else{
				$scope.hasfilesystem = false;
			}	
		}

		$scope.changeFs = function(){
			drawFileSystemUsage("filesystem-usage-chart");
		}

		function drawFileSystemUsage(elementId) {
			var titles = ["Time","Total Usage(总使用量)"];			
			var data = [];

			for (var i = 0; i < $scope.nodeInfos.data.length; i++) {
				var cur = $scope.nodeInfos.data[i];
				var filesystems = cur.filesystem;
				var index = _.map(filesystems,"device").indexOf($scope.selectedfs.value);
						
				var totalUsage = Math.floor((filesystems[index].usage * 10000.0) / filesystems[index].capacity) / 100.0;		
				
				if(data.length < 60){
					var elements = [];
					elements.push(cur.timestamp);
					elements.push(totalUsage);
					data.push(elements);
				}else{
					data[i].push(totalUsage);
				}
			}

			drawLineChart(titles, data, elementId, "%(百分比)");
		}
		
		//diskio
		function prepareDrawDiskIO(elementId){
			$scope.disks = [];
			if(_.isUndefined($scope.selecteddisk)){
				$scope.selecteddisk = {
					value : ""
				};
			}
			
			var currentdisks = $scope.nodeInfos.data[$scope.nodeInfos.data.length-1].diskio.io_service_bytes;
			for (var key in $scope.machineInfo.disk_map) {
				var thisdisk = _.find(currentdisks,function(disk){
					var value = disk.major + ":" + disk.minor;
					return key == value;
				});

				if(!_.isUndefined(thisdisk)){
					var disk = {
						"name" : $scope.machineInfo.disk_map[key].name,
						"value" : key
					}
					$scope.disks.push(disk);	
				}	
			}
			
			if($scope.disks.length > 0 && $scope.selecteddisk.value == ""){
				$scope.selecteddisk.value = $scope.disks[0].value;
			}

			if($scope.disks.length > 0){
				$scope.hasdiskio = true;
				drawDiskIO(elementId);
			}else{
				$scope.hasdiskio = false;
			}	
		}

		$scope.changeDisk = function(){
			drawDiskIO("diskio-chart");
		}

		function drawDiskIO(elementId) {
			var titles = ["Time","Read(读)","Write(写)"];
			var data = [];
			
			if (hasResource($scope.nodeInfos.data, "diskio")) {
				for (var i = 0; i < $scope.nodeInfos.data.length; i++) {
					var cur = $scope.nodeInfos.data[i];
					var disks = cur.diskio.io_service_bytes;
					var selectedDisk = _.find(disks,function(disk){
						var value = disk.major + ":" + disk.minor;
						return $scope.selecteddisk.value == value;
					})
						
					if(data.length < 60){
						var elements = [];
						elements.push(cur.timestamp);
						elements.push(selectedDisk.stats.Read / (1024 * 1024 * 8));
						elements.push(selectedDisk.stats.Write / (1024 * 1024 * 8));
						data.push(elements);		
					}else{
						data[i].push(selectedDisk.stats.Read / (1024 * 1024 * 8));
						data[i].push(selectedDisk.stats.Write / (1024 * 1024 * 8));
					}
				}
			}
			
			drawLineChart(titles, data, elementId, "MB(兆字节)");
		}

    	var initialize = function(){
			$scope.$watch('$storage.cluster', function() {
				if(navigator.appVersion.indexOf("Mac")!=-1){
					$scope.fontsize = "font-size:8px";
				}else{
					$scope.fontsize = "";
				}
				$scope.hascpu = true;
				$scope.hasmemory = true;
				$scope.hasnetwork = true;
				$scope.hasdiskio = true;
				$scope.hasfilesystem = true;

				window.charts = {};
				$scope.getClusterNodes();
				$scope.getClusterServices();
			},true);
		}
		initialize();
    }]);
});