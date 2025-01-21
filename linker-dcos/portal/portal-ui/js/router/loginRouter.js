define(["login"],
	function(app) {
		return app.run(['$rootScope', '$state', '$stateParams',
				function($rootScope, $state, $stateParams) {
					$rootScope.$state = $state;
					$rootScope.$stateParams = $stateParams
				}
			])
			.config(['$controllerProvider','$provide','$compileProvider','$filterProvider',
				function($controllerProvider,$provide, $compileProvider, $filterProvider) {
					app.controllerProvider = $controllerProvider;
					app.provide = $provide;
					app.compileProvider = $compileProvider;
                    app.filterProvider = $filterProvider;
				}
			])
			.config(['$translateProvider', function($translateProvider) {
				var lang = window.navigator.language.indexOf("zh") != -1 ? "zh" : "en";
				$translateProvider.useStaticFilesLoader({
					prefix: 'locales/',
					suffix: '.json'
				});
				$translateProvider.useLocalStorage();
				$translateProvider.useSanitizeValueStrategy('escape');
				$translateProvider.preferredLanguage(lang);

			}])
			.run(['$rootScope', '$translate',
				function($rootScope, $translate) {
					$rootScope.$translate = $translate;
				}
			])
			.config(['$stateProvider', '$urlRouterProvider', function($stateProvider, $urlRouterProvider) {
				$urlRouterProvider.otherwise('/');
				$stateProvider
					.state('login', {
						url: '/',
						templateUrl: 'templates/login/main.html',
						controller: 'LoginController',
						resolve: {
							loadCtrl: ["$q", function($q) {
								var deferred = $q.defer();
								require(["controllers/login/main"], function() {
									deferred.resolve();
								});
								return deferred.promise;
							}],
						}
					})
					.state('forgetPassword', {
						url: '/forgetPassword',
						templateUrl: 'templates/password/forget.html',
						controller: 'ForgetPasswordController',
						resolve: {
							loadCtrl: ["$q", function($q) {
								var deferred = $q.defer();
								require(["controllers/password/forget"], function() {
									deferred.resolve();
								});
								return deferred.promise;
							}],
						}
					})
					.state('setPassword', {
						url: '/setPassword',
						templateUrl: 'templates/password/set.html',
						controller: 'SetPasswordController',
						resolve: {
							loadCtrl: ['$q', function($q) {
								var deferred = $q.defer();
								require(['controllers/password/set'], function() {
									deferred.resolve();
								});
								return deferred.promise;
							}]
						}
					})
				
					
			}])
	})