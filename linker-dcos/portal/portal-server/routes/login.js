'use strict';
// var request = require('request');

var urlCfg = global.obj.urlCfg;
var logger = require('../utils/logger');

var providerUtil = require('../utils/providerUtil');
var ProviderUtil = new providerUtil("identityProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;


module.exports = function(app) {
    app.post('/user/login', ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
            url: ProviderUtil.rebuildUrl(global.obj.identity_url + urlCfg.login_api.login),
            method: 'POST',
            json: true,
            body: req.body
        };

        var callback = function(error, response, body) {
            if (error || response.statusCode >= 400) {
                logger.error('Error login:', error ? error.errno : response.statusCode, body);
                next(new ResponseError(error, response, body));
            } else {
                logger.trace('Login success', body);

                try {
                    body = JSON.parse(body);
                } catch(e) {
                    logger.error('Login JSON parse', body);
                    body = body;                   
                }

                logger.trace('Login body info', body)
                logger.trace('Login body typeof', typeof(body))
                logger.trace('Login req session', req.session);
                req.session.token = body.data.id;
                req.session.userid= body.data.userid;
                res.status(200).send(body);
            }
        };
        request(options, callback);
    });
    app.get('/logout', function(req, res) {
        req.session.destroy(function(err) {
            if (err) {
                res.status(500).send("Log out failed!");
            } else {
                res.status(200).send();
            }
        })

    });


};
