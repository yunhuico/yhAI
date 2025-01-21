define([
	'angular',
	'controllers/index',
	'services/index',
	'directives/index',
	], function(angular) {
	return angular.module("LinkerDCOS", ['ui.router', 'ui.bootstrap','ngStorage' ,'ngCookies','pascalprecht.translate',
        'app.controllers','app.services','app.directives'
        ]);
})
