'use strict';

var fs = require('fs');
var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;

var logger = require('../utils/logger');

var Authentication = require('../utils/authentication');

var providerUtil = require('../utils/providerUtil');
var ProviderUtil = new providerUtil("controllerProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;

module.exports = function(app) {
	//get key pairs
	app.get('/keypair', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.keypair + '?sort=-time_update&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get keypairs list:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get keypairs list', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//upload public key
	app.post('/keypair/upload', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		req.body.user_id = req.session.userid;
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.keypair),
			method: 'POST',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error upload keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('upload keypair', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//create key pair
	app.post('/keypair/create', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		req.body.user_id = req.session.userid;
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.keypair + "userSelfCreate"),
			method: 'POST',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error create keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('create keypair', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//download key pair
	app.get('/keypair/download/:userid', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.keypair + 'downLoadKey/' + req.params.userid),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error download keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Download keypair', body);
				res.set('Content-Type', 'application/octet-stream');
				res.status(200).send(response.body);
			}
		};
		request(options, callback);
	});

	//delete keypair
	app.delete('/keypair/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.keypair + req.params.id),
			method: 'DELETE',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error delete keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Delete keypair', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//get smtp servers
	app.get('/smtp', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.smtp + '?sort=-time_update&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get smtp servers list:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get smtp servers list', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//add smtp server
	app.post('/smtp', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.smtp),
			method: 'POST',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error add smtp server:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('add smtp server', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//update smtp server
	app.put('/smtp/:id', Authentication.ensureAuthenticated,ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.smtp + req.params.id),
          method: 'PUT',
          json:true,
          body: req.body,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error update smtp server:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Update smtp server', body);
               res.status(200).send(body);
          }
        };      
        request(options, callback);
    });

	//delete smtp server
	app.delete('/smtp/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.smtp + req.params.id),
			method: 'DELETE',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error delete smtp server:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Delete smtp server', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});

	//创建Provider
	app.post('/provider', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		req.body.user_id = req.session.userid;
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.provider),
			method: 'POST',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error upload keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('upload keypair', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});





	//获取provider列表
	app.get('/provider', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.provider + '?sort=-time_update&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get keypairs list:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get keypairs list', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	
	
	app.get('/provider/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		if (req.params.id == 'validate') {
			next();
		} else {
			var options = {
				url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.provider + req.params.id),
				method: 'GET',
				json: true,
				headers: {
					'X-Auth-Token': req.session.token
				}
			};
			var callback = function(error, response, body) {
				if (error || response.statusCode >= 400) {
					logger.error('Error get keypair :', error ? error.errno : response.statusCode, body);
					next(new ResponseError(error, response, body));
				} else {
					logger.trace('Get keypair list', body);
					res.status(200).send(body);
				}
			};
			request(options, callback);
		}
	});
	
	
	
	
	app.get('/provider/validate', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.provider + 'validate?provider_name=' + req.query.provider_name),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get provider name:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get provider name', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});


	//根据id编辑provider
	app.put('/provider/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.provider +  req.params.id),
			method: 'PUT',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error save user:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Save user', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	
	//根据id删除provider
	app.delete('/provider/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.provider + req.params.id),
			method: 'DELETE',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error delete keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Delete keypair', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	
	app.get('/dockerregistries', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.dockerregistries + '?sort=-time_update&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get keypairs list:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get keypairs list', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	app.post('/dockerregistries', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		req.body.user_id = req.session.userid;
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.dockerregistries),
			method: 'POST',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error save user:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Save user', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	app.delete('/dockerregistries/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.dockerregistries + req.params.id),
			method: 'DELETE',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error delete keypair:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Delete keypair', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	app.get('/registryValidate', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.dockerregistries +'/registryValidate?name=' + req.query.name+'&type='+req.query.type+'&user_id='+req.session.userid),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get keypairs list:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get keypairs list', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
};