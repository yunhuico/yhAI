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

module.exports = function(app) {
	app.get('/network', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.network + '?sort=time&count=true&internal='+ req.query.internal + '&skip=' + req.query.skip + '&limit=' + req.query.limit + '&cluster_id=' + req.query.cluster_id),
			method: 'GET',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error get network list:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Get network list', body);
				res.status(200).send(body);
			}
		};
		request(options, callback);
	});
	app.post('/network', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.network),
			method: 'POST',
			json: true,
			body: req.body,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		console.log(options);
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error creating network :', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Created network', body);
				res.status(200).send(body);
			}
		};
		logger.trace("Start to create network by request" + options.url);
		request(options, callback);
	});
	app.delete('/network/:networkid', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.network+req.params.networkid),
			method: 'DELETE',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error deleteing network:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Deleted network', body);
				res.status(200).send(body);
			}
		};
		logger.trace("Start to delete network by request " + options.url);
		request(options, callback);
	});
	app.delete('/network', Authentication.ensureAuthenticated, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.network+'?cluster_id='+req.query.cluster_id),
			method: 'DELETE',
			json: true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error deleteing all network:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Deleted all network', body);
				res.status(200).send(body);
			}
		};
		logger.trace("Start to delete all network by request " + options.url);
		request(options, callback);
	});
	app.get('/network/validate', Authentication.ensureAuthenticated,ProviderUtil.parseProviderUrl, function(req, res, next) {
		var options = {
			url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.network + '/validate?username='+req.query.username+'&networkname='+req.query.networkname),
			method: 'GET',
			json:true,
			headers: {
				'X-Auth-Token': req.session.token
			}
		};
		var callback = function(error, response, body) {
			if (error || response.statusCode >= 400) {
				logger.error('Error check network name:', error ? error.errno : response.statusCode, body);
				next(new ResponseError(error, response, body));
			} else {
				logger.trace('Check network name', body);
				res.status(200).send(body);
			}
		};
		logger.trace("Start to check network name " + options.url);
		request(options, callback);
	});
};