require.config({
	baseUrl: 'js',
	paths: {
		'domReady': 'libs/domReady',
		'angular': 'libs/angular.min',
		'uiRouter': 'libs/angular-ui-router.min',
		'lodash':'libs/lodash.min',
		'jquery': 'libs/jquery-1.11.3.min',
		'bootstrap': 'libs/bootstrap.min',
		'angular-bootstrap': 'libs/ui-bootstrap-tpls-0.14.3.min',
		'app': 'app',
		'router': 'router/router',
		'angular-cookies':'libs/angular-cookies.min',
		'angular-translate':'libs/angular-translate.min',
		'angular-translate-local':'libs/angular-translate-storage-local.min',
		'angular-translate-cookie':'libs/angular-translate-storage-cookie.min',
		'angular-translate-loader':'libs/angular-translate-loader-static-files.min',
		'ngStorage':'libs/ngStorage.min',
		// 'google-jsapi':'libs/google/google.jsapi',
		// 'google-uds':'libs/google/google.uds',
		// 'google':'libs/google/google.chart',
		'jit':'libs/jit-yc',
		'base64':'libs/base64.min'
	},
	shim: {
		'angular': {
			exports: 'angular'
		},
		'lodash': {
			exports: '_'
		},
		'angular-bootstrap': {
			deps: ['angular']
		},
		'angular-cookies' : {
            deps: ['angular']
		},
		'angular-translate' : {
            deps: ['angular']
		},
		'angular-translate-loader' : {
            deps: ['angular-translate']
		},
		'angular-translate-cookie' : {
            deps: ['angular-translate']
		},
		'angular-translate-local' : {
            deps: ['angular-translate','angular-translate-cookie']
		},
		'ngStorage':{
			deps:['angular']
		},
		'uiRouter': {
			deps: ['angular']
		},
		'bootstrap': {
			deps: ['jquery']
		},
		// 'google':{
		// 	deps: ['google-jsapi','google-uds']
		// }
	}
});
/**
 * bootstraps angular onto the window.document node
 */
define([
	'require','angular','app','lodash','jquery','uiRouter','angular-bootstrap',
	'bootstrap','router','angular-cookies','ngStorage','base64',
	'angular-translate','angular-translate-loader','angular-translate-cookie','angular-translate-local',
], function(require, angular, app,_,$) {
	'use strict';
	require(['domReady!'], function(document) {
		angular.bootstrap(document, ['LinkerDCOS']);
	});
});
