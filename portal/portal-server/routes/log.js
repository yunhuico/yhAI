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
    app.get('/logs', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.log + '?sort=-time_create&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit+"&cluster_id="+req.query.clusterid),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get log list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get log list', body);
               res.status(200).send(body);
          }
        };      
        request(options, callback);
    });
   

    
};