define(['app'], function(app) {
    'use strict';
    app.compileProvider.directive('servicegroupnode', ["$q", "$http","$cookies", function($q, $http,$cookies) {
        return {
            restrict : 'E',
            template    : '<div class="service-group" style="{{getZoomingCssString(\'node\')}}" data-toggle="context" data-groupid="{{node.id}}" data-grouptype="template">'+
                    '<div class="service-group-image-container" style="{{getZoomingCssString(\'service_group_image_container\')}}"><img src="{{node.data.imageSrc}}" class="container-image" draggable="false"/></div>'+
                    '<div class="a-app-item-id"><div class="a-app-item-id-label" style="{{getZoomingCssString(\'app_item_label\')}}">{{node.data.name}}</div></div>'+
                '</div>'
        }
    }]);

    app.compileProvider.directive('normalgroupnode', ["$q", "$http","$cookies", function($q, $http,$cookies) {
        return {
            restrict : 'E',
            template    : '<div style="text-align: center;">'+
                    '<div id="group_{{node.id}}" class="group" style="{{getZoomingCssString(\'group\')}}" data-groupid="{{node.id}}" data-grouptype="normalgroup">'+
                        '<div ng-repeat="app in node.data.apps" class="group-app-item" style="{{getZoomingCssString(\'group_app_item\')}}" data-appid="{{node.id}}/{{app.id}}" data-toggle="context">'+
                            '<div class="a-app-image-container" style="{{getZoomingCssString(\'app_image_container\')}}"><img src="{{app.imageSrc}}" class="container-image" draggable="false"/></div>'+
                            '<div class="a-app-item-id"><div class="a-app-item-id-label" style="<{{getZoomingCssString(\'app_item_label\')}}">{{app.id}}<span class="superscript" style="{{getZoomingCssString(\'superscript\')}}">{{app.instances}}</span></div></div>'+
                        '</div>'+
                        // '<span class="glyphicon glyphicon-record group-anchor" ng-if="node.data.apps.length>0 && allow_update_sg" title="{{\'rightContent.serviceDesign.linkDependency\' | translate}}"></span>'+
                    '</div>'+
                    '<div style="cursor:auto；{{getZoomingCssString(\'group_label\')}}">'+
                        '<span data-toggle="context" data-groupid="{{node.id}}" data-grouptype="normalgroup" class="group-label">{{node.data.name}}</span>'+
                        '<span class="glyphicon glyphicon-search" ng-if="checkIfComponentsHaveDependencies(node.data)" title="{{\'service.componentrelations\' | translate}}" style="float:right"></span>'+
                    '</div>'+
                '</div>'
        }
    }]);

    app.compileProvider.directive('singleappnode', ["$q", "$http","$cookies", function($q, $http,$cookies) {
        return {
            restrict : 'E',
            template    : '<div class="group-app-item" style="{{getZoomingCssString(\'group_app_item\')}}" data-appid="{{node.id}}" data-toggle="context">'+
                            '<div class="a-app-image-container" style="{{getZoomingCssString(\'app_image_container\')}}"><img src="{{node.data.imageSrc}}" class="container-image" draggable="false"/></div>'+
                            '<div class="a-app-item-id"><div class="a-app-item-id-label" style="<{{getZoomingCssString(\'app_item_label\')}}">{{idToSimple(node.data.name)}}<span class="superscript" style="{{getZoomingCssString(\'superscript\')}}">{{node.data.instances}}</span></div></div>'+
                        '</div>'
        }
    }]);
});