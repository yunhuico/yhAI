'use strict';
// var request = require('request');

var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;

var logger = require('../utils/logger');

var Authentication = require('../utils/authentication');

var providerUtil = require('../utils/providerUtil');

var ProviderUtil = new providerUtil("identityProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;

module.exports = function(app) {
    app.get('/user', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.get + '?sort=-time_update&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit),
            method: 'GET',
            json: true,
            headers: {
                'X-Auth-Token': req.session.token
            }
        };
        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error get user list:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Get user list', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });
    app.post('/user', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.get),
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
    app.put('/user/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.get + req.params.id),
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

    //validate whether username is duplicated
    app.get('/user/validate', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.validate + "?username=" + req.query.username),
            method: 'GET',
            json: true,
            headers: {
                'X-Auth-Token': req.session.token
            }
        };
        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error validate user:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Validated user', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });
    app.delete('/user/:id', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.get + req.params.id),
            method: 'DELETE',
            json: true,
            headers: {
                'X-Auth-Token': req.session.token
            }
        };
        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error delete user:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Delete user', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });
    app.get('/user/profile', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.get + req.session.userid),
            method: 'GET',
            json: true,
            headers: {
                'X-Auth-Token': req.session.token
            }
        };
        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error get user information:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Get user information', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });
    app.put('/user/profile/changepassword', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.user_api.password + req.session.userid),
            method: 'PUT',
            json: true,
            headers: {
                'X-Auth-Token': req.session.token
            },
            body: req.body
        };
        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error update user password:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Updated user password', body);
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });

};
