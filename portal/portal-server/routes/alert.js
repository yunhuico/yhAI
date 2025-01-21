'use strict';

var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;

var logger = require('../utils/logger');

var Authentication = require('../utils/authentication');

var providerUtil = require('../utils/providerUtil');

var ProviderUtil = new providerUtil("controllerProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;

module.exports = function (app) {
  app.get('/alerts', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
    var options = {
      url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.alert) + '?count=true' + '&skip=' + req.query.skip + '&limit=' + req.query.limit + '&alert_name=' + req.query.alert_name + '&action=' + req.query.action,
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };
    console.log(options.url);
    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400) {
        logger.error('Error get alert list:', error ? error.errno : response.statusCode, body);
        next(new ResponseError(error, response, body));
      } else {
        logger.trace('Get alert list', body);
        res.status(200).send(body);
      }
    };      
    request(options, callback);
  });
};