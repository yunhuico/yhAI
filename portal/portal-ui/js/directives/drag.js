define(['app'], function(app) {
	'use strict';
	app.compileProvider.directive('drag', [function() {
		return {
			restrict: 'A',
			link: function(scope, eles, attrs) {
				if(window.File && window.FileList && window.FileReader && window.Blob) {
					eles.bind('dragover', function(e) {
						e.stopPropagation();
						e.preventDefault();
						eles.addClass('hover');
					});
					
					eles.bind('dragleave', function(e) {
						e.stopPropagation();
						e.preventDefault();
						eles.removeClass('hover');
					});
					
					eles.bind('drop', function(e) {
						e.preventDefault();
	        				e.stopPropagation();
						eles.removeClass('hover');
						var files;
						if(e.originalEvent) {
							files = e.originalEvent.dataTransfer.files
						}else if(e.dataTransfer) {
							files = e.dataTransfer.files
						}
						if(files.length>1) {
							scope.$parent.showLengthWarning = true;
							scope.$parent.showTypeWarning = false;
							scope.$parent.service['group'] = '';
							scope.$apply();
						}else if(files[0].name.toLowerCase().indexOf('.json') !== -1) {
							var reader = new FileReader();
							reader.onload = function() {
								scope.$parent.service['group'] = this.result;
								scope.$parent.showLengthWarning = false;
								scope.$parent.showTypeWarning = false;
								scope.$apply();
							}
							reader.readAsText(files[0]);
						}else if(files[0].name.toLowerCase().indexOf('.json') === -1) {
							scope.$parent.showLengthWarning = false;
							scope.$parent.showTypeWarning = true;
							scope.$parent.service['group'] = '';
							scope.$apply();
						}
					})
			    }
				
			}
		}
	}])
})
