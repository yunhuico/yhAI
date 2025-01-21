define([
	'angular',
	'controllers/index',
	'services/index',
	'directives/index'
	], function(angular) {
	return angular.module("Login", ['ui.router', 'ui.bootstrap', 'ngCookies','pascalprecht.translate',
        'app.controllers','app.services','app.directives'
        ]);
})