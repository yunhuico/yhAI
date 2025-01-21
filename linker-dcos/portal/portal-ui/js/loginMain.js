require.config({
	baseUrl: 'js',
	paths: {
		'domReady': 'libs/domReady',
		'angular': 'libs/angular.min',
		'uiRouter': 'libs/angular-ui-router.min',
		'underscore':'libs/underscore.min',
		'jquery': 'libs/jquery-1.11.3.min',
		'bootstrap': 'libs/bootstrap.min',
		'angular-bootstrap': 'libs/ui-bootstrap-tpls-0.14.3.min',
		'login': 'loginApp',
		'loginRouter': 'router/loginRouter',
		'angular-cookies':'libs/angular-cookies.min',
		'angular-translate':'libs/angular-translate.min',		
		'angular-translate-local':'libs/angular-translate-storage-local.min',
		'angular-translate-cookie':'libs/angular-translate-storage-cookie.min',
		'angular-translate-loader':'libs/angular-translate-loader-static-files.min'

	},
	shim: {
		'angular': {
			exports: 'angular'
		},
		'underscore': {
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
		'uiRouter': {
			deps: ['angular']
		},
		'bootstrap': {
			deps: ['jquery']
		}
	}
});
/**
 * bootstraps angular onto the window.document node
 */
define([
	'require','angular','login','underscore','jquery','uiRouter','angular-bootstrap',
	'bootstrap','loginRouter','angular-cookies',
	'angular-translate','angular-translate-loader','angular-translate-cookie','angular-translate-local'
], function(require, angular, login,_,$) {
	'use strict';
	require(['domReady!'], function(document) {
		angular.bootstrap(document, ['Login']);
		
	});
});
