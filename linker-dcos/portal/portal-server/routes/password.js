'use strict';
// var request = require('request');

var urlCfg = global.obj.urlCfg;
var logger = require('../utils/logger');

var providerUtil = require('../utils/providerUtil');
var ProviderUtil = new providerUtil("identityProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;


module.exports = function(app) {
    app.post('/forget', ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + '/v1/user' + urlCfg.dcosclient_api.forget + '?username=' + req.body.username + '&ip=' + req.body.ip),
            method: 'PUT',
            json: true
        };

        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error password:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Success search password', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });
    app.post('/forget/change', ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + '/v1/user' + urlCfg.dcosclient_api.reset + '?username=' + req.body.username + '&activecode=' + req.body.activecode + '&newpassword=' + req.body.newpassword),
            method: 'PUT',
            json: true
        };

        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error reset password:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Success reset password', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });


};