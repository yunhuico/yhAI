'use strict';
// var request = require('request');

var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;

var logger = require('../utils/logger');

var Authentication = require('../utils/authentication');

var providerUtil = require('../utils/providerUtil');

var ProviderUtil = new providerUtil("dcosClientProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;

module.exports = function (app) {
	app.get('/package/search/:query?', Authentication.ensureAuthenticated, function(req, res, next) {
		var header = {};
		var body_content = {};
		var url, content_type;
		if(req.params.query) {
			content_type = 'search'
			body_content = {'query': req.params.query};
		}else {
			content_type = 'search';
			body_content = req.body;
		}
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.package+"/"+content_type),
			method: "POST",
			json: true,
			headers: {
    				'X-Auth-Token': req.session.token,
    				'Content-Type': 'application/vnd.dcos.package.'+content_type+'-request+json;charset=utf-8;version=v1',
    				'Accept': 'application/vnd.dcos.package.'+content_type+'-response+json;charset=utf-8;version=v1'
    			},
			body: body_content
		};
		var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error search framework:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
          	   var packages = body.packages;
               for(var i=0; i<packages.length; i++) {
			 	   if(packages[i]['images']) {
					   var str = packages[i]['images']['icon-medium'].replace(/master.mesos/, req.query.clientAddr);
					   packages[i]['images']['icon-medium'] = str;
				   }
               }
               logger.trace('search framework:', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
	});

	app.post('/package/describe', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.package+"/describe"),
			method: "POST",
			json: true,
			headers: {
				'X-Auth-Token': req.session.token,
				'Content-Type': 'application/vnd.dcos.package.describe-request+json;charset=utf-8;version=v1',
				'Accept': 'application/vnd.dcos.package.describe-response+json;charset=utf-8;version=v1'
			},
			body: req.body
		};
		var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error describe framework:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
          	   var pkg = body.resource;
			   if(pkg['images']) {
				   var str = pkg['images']['icon-medium'].replace(/master.mesos/, req.query.clientAddr);
				   pkg['images']['icon-medium'] = str;
			   }
               logger.trace('Describe framework:', body);
               res.status(200).send(body);
          }
        };
		request(options, callback);
	});

	app.get('/package/list', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.package+"/list"),
			method: "POST",
			json: true,
			headers: {
				'X-Auth-Token': req.session.token,
				'Content-Type': 'application/vnd.dcos.package.list-request+json;charset=utf-8;version=v1',
				'Accept': 'application/vnd.dcos.package.list-response+json;charset=utf-8;version=v1'
			},
			body: req.body
		};
		var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error list framework:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('List framework:', body);
               res.status(200).send(body);
          }
        };
		request(options, callback);
	});

	app.post('/package/install', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.package+"/install"),
			method: "POST",
			json: true,
			headers: {
				'X-Auth-Token': req.session.token,
				'Content-Type': 'application/vnd.dcos.package.install-request+json;charset=utf-8;version=v1',
				'Accept': 'application/vnd.dcos.package.install-response+json;charset=utf-8;version=v1'
			},
			body: req.body
		};
		var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error install framework:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Install framework:', body);
               res.status(200).send(body);
          }
        };
		request(options, callback);
	});

	app.post('/package/uninstall', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.package+"/uninstall"),
			method: "POST",
			json: true,
			headers: {
				'X-Auth-Token': req.session.token,
				'Content-Type': 'application/vnd.dcos.package.uninstall-request+json;charset=utf-8;version=v1',
				'Accept': 'application/vnd.dcos.package.uninstall-response+json;charset=utf-8;version=v1'
			},
			body: req.body
		};
		var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error uninstall framework:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Uninstall framework:', body);
               res.status(200).send(body);
          }
        };
		request(options, callback);
	});

    app.get('/tasks', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.framework +'/tasks?sort=-time_update&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit + '&host_ip=' + req.query.host_ip),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get framework tasks list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get framework tasks list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });

    app.get('/package/repository/list', Authentication.ensureAuthenticated, function(req, res, next) {
    		var options = {
    			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.repository + "/list"),
    			method: 'POST',
    			json: true,
    			headers: {
    				'X-Auth-Token': req.session.token,
    				'Content-Type': 'application/vnd.dcos.package.repository.list-request+json;charset=utf-8;version=v1',
    				'Accept': 'application/vnd.dcos.package.repository.list-response+json;charset=utf-8;version=v1'
    			},
    			body: req.body
    		};
    		var callback = function(error, response, body) {
         	if(error || response.statusCode >= 400){
               	logger.error('Error get package repository list:', error ? error.errno : response.statusCode, body);
               	next(new ResponseError(error, response, body));
          	}else{
            		logger.trace('Get package repository list', body);
               	res.status(200).send(body);
          	}
        };
        request(options, callback);
    });

    app.delete('/package/repository/:name', Authentication.ensureAuthenticated, function(req, res, next) {
    		var options = {
    			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.repository + '/delete'),
    			method: 'POST',
    			json: true,
    			headers: {
    				'X-Auth-Token': req.session.token,
    				'Content-Type': 'application/vnd.dcos.package.repository.delete-request+json;charset=utf-8;version=v1',
    				'Accept': 'application/vnd.dcos.package.repository.delete-response+json;charset=utf-8;version=v1'
    			},
    			body: {
    				'name': req.params.name
    			}
    		};
    		var callback = function(error, response, body) {
         	if(error || response.statusCode >= 400){
               	logger.error('Error get package repository delete:', error ? error.errno : response.statusCode, body);
               	next(new ResponseError(error, response, body));
          	}else{
               	logger.trace('Get package repository delete', body);
               	res.status(200).send(body);
          	}
        };
        request(options, callback);
    });

    app.post('/package/repository/add', Authentication.ensureAuthenticated, function(req, res, next) {
    		var options = {
    			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.repository + '/add'),
    			method: 'POST',
    			json: true,
    			headers: {
    				'X-Auth-Token': req.session.token,
    				'Content-Type': 'application/vnd.dcos.package.repository.add-request+json;charset=utf-8;version=v1',
    				'Accept': 'application/vnd.dcos.package.repository.add-response+json;charset=utf-8;version=v1'
    			},
    			body: req.body
    		};
    		var callback = function(error, response, body) {
         	if(error || response.statusCode >= 400){
               	logger.error('Error get package repository add:', error ? error.errno : response.statusCode, body);
               	next(new ResponseError(error, response, body));
          	}else{
               	logger.trace('Get package repository add', body);
               	res.status(200).send(body);
          	}
        };
        request(options, callback);
    })
};