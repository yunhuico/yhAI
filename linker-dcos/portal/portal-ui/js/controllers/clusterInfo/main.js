define(['app', 'services/clusterInfo/main', 'services/node/main','services/monitor/main'], function(app) {
    'use strict';
    app.controllerProvider.register('ClusterInfoController', ['$scope', '$localStorage', '$uibModal', '$location', '$state', 'ResponseService', 'MonitorService', 'CommonService', 'ClusterInfoService', '$q',
    function($scope, $localStorage, $uibModal, $location, $state, ResponseService,MonitorService,CommonService, ClusterInfoService, $q) {
			$scope.$storage = $localStorage;
      // window.charts = {};

			////////////////////////////////////////////
			// 							getClusterNodes   				//
			////////////////////////////////////////////

    	$scope.getClusterNodes = function(){
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

			$scope.changeNode = function(){
				clearLocalSelectedValue();

				if($location.path() == "/clusterInfo"){
          // $scope.nodeMonitor();
          $scope.cpuMonitor();
          $scope.memMonitor();
          $scope.diskMonitor();
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

      function hasMonitorData() {
        return $scope.hascpu || $scope.hasmemory;
      }

      $scope.cpuMonitor = function() {
        clearTimeout(window.reloadCPUMonitorAction);

        if ( $location.path() == "/clusterInfo" ) {
          $q.all([
            $scope.getDataCPU(),
						$scope.getPredictDataCPU(),
            $scope.getThresholdCpu()
          ]).then(function() {
    				$scope.hascpu = true;
            $scope.hasMonitorData = hasMonitorData();
            $scope.hasLoadingCpu = false;
            drawCpuTotalUsage("cpu-total-usage-chart");
            window.reloadCPUMonitorAction = setTimeout($scope.cpuMonitor,30000);
          }, function( error ) {
            console.log( 'get cpu data error', error );
    				// $scope.hascpu = false;
            $scope.hasMonitorData = hasMonitorData();
            window.reloadCPUMonitorAction = setTimeout($scope.cpuMonitor,30000);
          });
        }
      }

      $scope.memMonitor = function() {
        clearTimeout(window.reloadMemMonitorAction);

        if ( $location.path() == "/clusterInfo" ) {
					$scope.getAlertMem();
          $q.all([
            $scope.getDataMem(),
						$scope.getPredictDataMem(),
            $scope.getThresholdMem()
          ]).then(function() {
    				$scope.hasmemory = true;
            $scope.hasMonitorData = hasMonitorData();
            $scope.hasLoadingMemory = false;
            drawMemoryUsage("memory-usage-chart");
            window.reloadMemMonitorAction = setTimeout($scope.memMonitor,30000);
          }, function( error ) {
            console.log( 'get memory data error', error );
    				// $scope.hasmemory = false;
            $scope.hasMonitorData = hasMonitorData();
            window.reloadMemMonitorAction = setTimeout($scope.memMonitor,30000);
          });
        }
      }

      $scope.diskMonitor = function() {
        clearTimeout(window.reloadDiskMonitorAction);

        if ( $location.path() == "/clusterInfo" ) {
          $q.all([
            $scope.getDiskusage()
          ]).then(function() {
            // console.log($scope.totalData.disk);
    				$scope.hasdiskusage = true;
            $scope.hasMonitorData = hasMonitorData();
            $scope.hasLoadingDisk = false;
            drawDiskUsage("disk-usage-chart");
            window.reloadDiskMonitorAction = setTimeout($scope.diskMonitor,30000);
          }, function( error ) {
            console.log( 'get memory data error', error );
    				// $scope.hasmemory = false;
            $scope.hasMonitorData = hasMonitorData();
            window.reloadDiskMonitorAction = setTimeout($scope.diskMonitor,30000);
          });
        }
      }

      $scope.getDataCPU = function () {
        var deferred = $q.defer();

  			return ClusterInfoService.getTotalCpu()
  			.then(function (data) {
  				if (! data.data)
  					return;

          var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}

          $scope.totalData.cpu = result;
          deferred.resolve();
          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

      $scope.getDataMem = function () {
        var deferred = $q.defer();

  			return ClusterInfoService.getTotalMem()
  			.then(function (data) {
  				if (! data.data)
  					return;

  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}

          $scope.totalData.mem = result;
          deferred.resolve();
          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

      $scope.getThresholdCpu = function () {
        var deferred = $q.defer();

  			return ClusterInfoService.getThresholdCpu()
  			.then(function (data) {
  				if (! data.data)
  					return;

  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}

  				$scope.totalData.threshold.cpu = result;

          deferred.resolve();
          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

      $scope.getThresholdMem = function () {
        var deferred = $q.defer();

  			return ClusterInfoService.getThresholdMem()
  			.then(function (data) {
  				if (! data.data)
  					return;
  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}

  				$scope.totalData.threshold.mem = result;

          deferred.resolve();
          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

      $scope.getDiskusage = function () {
        var deferred = $q.defer();

  			return ClusterInfoService.getDiskusage()
  			.then(function (data) {
  				if (! data.data)
  					return;
  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}

  				$scope.totalData.disk = result;

          deferred.resolve();
          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

			$scope.getTime = function( timestamp ) {
				if ( timestamp && timestamp != '' ) {
					var time = new Date( timestamp * 1000 );
					return time.getFullYear() + '-' + time.getMonth() + '-' + time.getDate() + ' ' +
								time.getHours() + ':' + time.getMinutes();
				}
			}

			$scope.getAlertMem = function () {
        var deferred = $q.defer();

  			return ClusterInfoService.getAlertMem()
  			.then(function (data) {
  				if (! data.data)
  					return;
  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}

					$scope.hasAlarm = Array.isArray( result );
  				$scope.alarm.mem = result;

          deferred.resolve();
          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

      var _predictData = {
  			cpu: [],
  			mem: []
  		};
  		$scope.getPredictDataCPU = function () {
				var deferred = $q.defer();

  			return ClusterInfoService.getPredictCpu()
  			.then(function (data) {
  				if (! data.data)
  					return;
  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}
  				_predictData.cpu = result;

					deferred.resolve();

          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};

  		$scope.getPredictDataMem = function () {
				var deferred = $q.defer();

  			return ClusterInfoService.getPredictMem()
  			.then(function (data) {
  				if (! data.data)
  					return;
  				var result;
  				try {
  					result = JSON.parse(data.data.replace(/'/g, '"'));
  				} catch (e) {
  					result = data.data;
  				}
  				_predictData.mem = result;

					deferred.resolve();

          return deferred.promise;
  			}, function( error ) {
          deferred.reject( error );
          return deferred.promise;
        })
  		};


      // Draw the graph for CPU usage.
  		function drawCpuTotalUsage(elementId) {
  			var titles = ["Time","Total Usage", "Prediction", "Threshold"];
  			var data = [];

  			if ( $scope.totalData.cpu ) {
  				var lastTime = 0;
          var thresholdArray = $scope.totalData.threshold.cpu;
  				for ( var i = 0 ; i < $scope.totalData.cpu.length ; i++ ) {
  					var cur = $scope.totalData.cpu[i];
            var cpuUsage = cur.Value * 100;
						var elements = [];
						elements.push(cur.Time);
						elements.push(cpuUsage);
						//the predict data will be empty in usage range.
					  elements.push(null);
						elements.push(null);
						data.push(elements);
  				}

  				data = drawUsage(_predictData.cpu, data);
					data = drawThresholdUsage( thresholdArray, data );
  			}

  			drawLineChart(titles, data, elementId, "Percentage (%)");
  		}

  		function drawMemoryUsage(elementId) {
  			var titles = ["Time","Total Usage", "Prediction", "Threshold"];
  			var data = [];
  			var lastTime = 0;
        var thresholdArray = $scope.totalData.threshold.mem;

  			if ( $scope.totalData.mem ) {
  				for (var i = 0 ; i < $scope.totalData.mem.length ; i++ ) {
  					var cur = $scope.totalData.mem[i];
            var totalMemory = cur.Value * 100;
						var elements = [];
						elements.push(cur.Time);
						elements.push(totalMemory);
						elements.push(null);
						elements.push(null);
						data.push(elements);
  				}

    			data = drawUsage(_predictData.mem, data);
					data = drawThresholdUsage( thresholdArray, data );
  			}

  			drawLineChart(titles, data, elementId, "Percentage (%)");
  		}

      function drawDiskUsage( elementId ) {
        var data = [['Node', 'Available', { role: "style" }, {'type': 'string', 'role': 'tooltip', 'p': {'html': true}}]];
        for ( var i = 0 ; i < $scope.totalData.disk.length ; i++ ) {
          var cur = $scope.totalData.disk[i];
					var availableDate = new Date( cur.date );
					var dateString = availableDate.getFullYear() + '-' + availableDate.getMonth() + '-' + availableDate.getDate() + ' ' +
													 availableDate.getHours() + ':' + availableDate.getMinutes();
					var tooltipHtml = '<div style="padding: 15px; white-space: nowrap;">' +
															'<div><b>' + cur.ID.substring( 0, 8 ) + '</b></div><br>' +
															'<div>Available: <b>' + (cur.available * 100).toFixed(2) + '</b></div>' +
															'<div>Time of available to 20%: <b>' + dateString + '</b></div>' +
														'</div>';
          if ( cur.available < 0.5 )
            data.push([ cur.ID.substring( 0, 8 ), cur.available * 100, 'Crimson', tooltipHtml ]);
          else
            data.push([ cur.ID.substring( 0, 8 ), cur.available * 100, 'royalblue', tooltipHtml ]);
        }

        var dataTable = new google.visualization.arrayToDataTable( data );

        var opts = {
          title: 'Available disk usage of node',
          hAxis: {
            viewWindow: {
              max: 100,
              min: 0
            }
          },
  				legend: {
  					position: 'bottom'
  				},
					tooltip: {
						isHtml: true
					}
        };

        if (!(elementId in window.charts))
  				window.charts[elementId] = new google.visualization.BarChart(document.getElementById(elementId));

        window.charts[elementId].draw(dataTable, opts);
      }

      function drawUsage(typeNodes, data) {
        var lastTime = data[data.length - 1][0];
        var lengthOfTypeNodes = typeNodes.length > 100 ? typeNodes.length : 180;
  			for ( var i = 0 ; i < lengthOfTypeNodes ; i++ ) {
  				// every 10 seconds, have a predict data.
  				// var time = lastTime.valueOf() + (i + 1) * 10000;
  				var val;
          var time;
          // @TODO after prediction data exisit, it need update
          if ( typeNodes.length > 100 ) {
            val = (typeNodes.length > 0 ? typeNodes[i].Value : 0) * 100;
            time = typeNodes[i].Time;
          } else {
            val = 0;
            time = lastTime + ( i + 1 ) * 10;
          }
          // @TODO after prediction data exisit, it need update
  				var elements = [];

  				elements.push(time);
  				elements.push(null);
  				elements.push(val);
					elements.push(null);
  				data.push(elements);
  			}

  			return data;
  		}

			function drawThresholdUsage( thresholdArray, data ) {
  			for ( var i = 0 ; i < thresholdArray.length ; i++ ) {
					var cur = thresholdArray[i];
					var val = cur.Value ? cur.Value * 100 : null;
  				var elements = [];

  				elements.push(cur.Time);
  				elements.push(null);
  				elements.push(null);
					elements.push(val);
  				data.push(elements);
  			}

  			return data;
			}

      // Draw a line chart.
  		function drawLineChart(seriesTitles, data, elementId, unit) {
  			var min = Infinity;
  			var max = -Infinity;
  			for (var i = 0; i < data.length; i++) {
  				// Convert the first column to a Date.
  				if (data[i] != null) {
  					data[i][0] = new Date(data[i][0] * 1000);
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

  			// console.log(data);
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
              max: 100,
  						min: 0
  					}
  				},
  				legend: {
  					position: 'bottom'
  				}
  			};

  			window.charts[elementId].draw(dataTable, opts);
  		}

      // Checks if the specified stats include the specified resource.
  		function hasResource(stats, resource) {
  			return stats.length > 0 && stats[0][resource];
  		}

      // Gets the length of the interval in nanoseconds.
  		function getInterval(current, previous) {
  			var cur = new Date(current);
  			var prev = new Date(previous);

  			// ms -> ns.
  			return (cur.getTime() - prev.getTime()) * 1000000;
  		}

      $scope.historys = [];
      var pushScaleHitory = function () {
        // console.log('run data');
        var flag = Math.floor(Math.random() * (100 - 0 + 1)) + 0;
        if (flag > 80) {
          $scope.historys.push({
            status: "extend",
            message: "Cluster extend 1 more node"
          });

        } else {
          $scope.historys.push({
            status: "ok",
            message: "Cluster is ok"
          });

        }

        $scope.$apply();

        if ($scope.historys.length > 7) {
          $scope.historys.shift();
        }
      };

      setInterval(function () {
        pushScaleHitory();
      }, 3000);


      $scope.$watch('$storage.cluster', function() {
				if ( navigator.appVersion.indexOf( "Mac" ) != -1 ) {
					$scope.fontsize = "font-size:8px";
				} else {
					$scope.fontsize = "";
				}
        $scope.hasMonitorData = true;
				$scope.hascpu = true;
				$scope.hasmemory = true;
        $scope.hasdiskusage = true;

        $scope.hasLoadingCpu = true;
        $scope.hasLoadingMemory = true;
        $scope.hasLoadingDisk = true;
        $scope.totalData = {};
        $scope.totalData.threshold = {};

				$scope.alarm = {};

				window.charts = {};
				$scope.getClusterNodes();
			},true);
  }]);
});
