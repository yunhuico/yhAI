define(["app"],
    function (app) {
        return app.run(['$rootScope', '$state', '$stateParams',
            function ($rootScope, $state, $stateParams) {
                $rootScope.$state = $state;
                $rootScope.$stateParams = $stateParams
            }
        ])
            .config(['$controllerProvider', '$provide', '$compileProvider', '$filterProvider',
                function ($controllerProvider, $provide, $compileProvider, $filterProvider) {
                    app.controllerProvider = $controllerProvider;
                    app.provide = $provide;
                    app.compileProvider = $compileProvider;
                    app.filterProvider = $filterProvider;
                }
            ])
            .config(['$translateProvider', function ($translateProvider) {
                var lang = window.navigator.language.indexOf("zh") != -1 ? "zh" : "en";
                $translateProvider.useStaticFilesLoader({
                    prefix: 'locales/',
                    suffix: '.json'
                });
                $translateProvider.useLocalStorage();
                $translateProvider.useSanitizeValueStrategy('escape');
                // $translateProvider.fallbackLanguage(lang);
                $translateProvider.preferredLanguage(lang);
                // $translateProvider.useMessageFormatInterpolation();

            }])
            .run(['$rootScope', '$translate',
                function ($rootScope, $translate) {
                    $rootScope.$translate = $translate;
                }
            ])
            .run(['$rootScope', '$cookies', '$http', 'ResponseService', function ($rootScope, $cookies, $http, ResponseService) {
                $rootScope.$on('$stateChangeStart', function (event, toState) {
                    if (!$cookies.get("username")) {
                        window.location = "/login.html";
                    }
                    if (toState.name === 'user' || toState.name === 'smtp') {
                        var request = {
                            "url": "/user/profile",
                            "method": "GET"
                        };

                        $http(request).success(function (data) {
                            if (data.data.rolename !== "sysadmin") {
                                event.preventDefault();
                                $rootScope.$state.go("node");
                                ResponseService.errorResponse({ "code": 401 });
                            }
                        }).error(function (error) {
                            ResponseService.errorResponse(error);
                        });
                    }

                })
            }])
            .config(['$stateProvider', '$urlRouterProvider', function ($stateProvider, $urlRouterProvider) {
                $urlRouterProvider.otherwise('/node');
                $stateProvider
                    .state('profile', {
                        url: "/profile",
                        templateUrl: 'templates/profile/main.html',
                        controller: 'ProfileController',
                        resolve: {
                            loadCtrl_user: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/profile/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }
                    })
                    .state('user', {
                        url: "/user",
                        templateUrl: 'templates/user/main.html',
                        controller: 'UserController',
                        resolve: {
                            loadCtrl_user: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/user/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }
                    })
                    .state('header', {
                        abstract: true,
                        templateUrl: 'templates/common/header.html',
                        controller: 'ClusterController',
                        resolve: {
                            loadCtrl_parent: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/cluster/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }
                    })
                    .state('clusterInfo', {
                        parent: 'header',
                        url: '/clusterInfo',
                        views: {
                            'main': {
                                templateUrl: 'templates/clusterInfo/main.html',
                                controller: 'ClusterInfoController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/clusterInfo/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('node', {
                        parent: 'header',
                        url: '/node',
                        views: {
                            'main': {
                                templateUrl: 'templates/node/main.html',
                                controller: 'NodeController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/node/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('nodedetail', {
                        parent: 'node',
                        url: '/{nodeid}',
                        views: {
                            'main@header': {
                                templateUrl: 'templates/node/detail.html',
                                controller: 'NodeDetailController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/node/detail"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('service', {
                        parent: 'header',
                        url: '/service',
                        views: {
                            'main': {
                                templateUrl: 'templates/service/main.html',
                                controller: 'ServiceController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/service/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('service.detail', {
                        url: '/{serviceName}?serviceStatus',
                        views: {
                            'main@header': {
                                templateUrl: 'templates/service/detail.html',
                                controller: 'ServiceDetailController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/service/detail"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('service.detail.container', {
                        url: '/{containerName}',
                        params: {
                            containerId: null
                        },
                        views: {
                            'main@header': {
                                templateUrl: 'templates/service/container.html',
                                controller: 'ContainerController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/service/container"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('network', {
                        parent: 'header',
                        url: '/network',
                        views: {
                            'main': {
                                templateUrl: 'templates/network/main.html',
                                controller: 'NetworkController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/network/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('framework', {
                        parent: 'header',
                        url: '/framework',
                        views: {
                            'main': {
                                templateUrl: 'templates/framework/main.html',
                                controller: 'FrameworkController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/framework/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('framework.detail', {
                        url: '/{frameworkName}',
                        views: {
                            'main@header': {
                                templateUrl: 'templates/framework/detail.html',
                                controller: 'FrameworkDetailController',
                                resolve: {
                                    loadCtrl: ['$q', function ($q) {
                                        var deferred = $q.defer();
                                        require(['controllers/framework/detail'], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }]
                                }
                            }
                        },
                        params: {
                            frameworkName: null,
                            packageVersion: null
                        }
                    })
                    .state('log', {
                        parent: 'header',
                        url: '/log',
                        views: {
                            'main': {
                                templateUrl: 'templates/log/main.html',
                                controller: 'LogController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/log/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('monitor', {
                        parent: 'header',
                        url: '/monitor',
                        views: {
                            'main': {
                                templateUrl: 'templates/monitor/main.html',
                                controller: 'MonitorController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/monitor/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('alert', {
                        parent: 'header',
                        url: '/alert',
                        views: {
                            'main': {
                                templateUrl: 'templates/alert/main.html',
                                controller: 'AlertController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/alert/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('component', {
                        parent: 'header',
                        url: '/component',
                        views: {
                            'main': {
                                templateUrl: 'templates/component/main.html',
                                controller: 'ComponentController',
                                resolve: {
                                    loadCtrl: ["$q", function ($q) {
                                        var deferred = $q.defer();
                                        require(["controllers/component/main"], function () {
                                            deferred.resolve();
                                        });
                                        return deferred.promise;
                                    }],
                                }
                            }
                        }
                    })
                    .state('keypair', {
                        url: '/configuration/key',
                        templateUrl: 'templates/configuration/main.html',
                        controller: 'ConfigController',
                        resolve: {
                            loadCtrl: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/configuration/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }

                    })
                    .state('smtp', {
                        url: '/configuration/smtp',
                        templateUrl: 'templates/configuration/main.html',
                        controller: 'ConfigController',
                        resolve: {
                            loadCtrl: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/configuration/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }

                    })
                    .state('platform', {
                        url: '/configuration/platform',
                        templateUrl: 'templates/configuration/main.html',
                        controller: 'ConfigController',
                        resolve: {
                            loadCtrl: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/configuration/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }

                    })
                    .state('registry', {
                        url: '/configuration/registry',
                        templateUrl: 'templates/configuration/main.html',
                        controller: 'ConfigController',
                        resolve: {
                            loadCtrl: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/configuration/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }

                    })
                    .state('information', {
                        url: '/information',
                        templateUrl: 'templates/information/main.html',
                        controller: 'InformationController',
                        resolve: {
                            loadCtrl: ["$q", function ($q) {
                                var deferred = $q.defer();
                                require(["controllers/information/main"], function () {
                                    deferred.resolve();
                                });
                                return deferred.promise;
                            }],
                        }

                    })
            }])
    })
