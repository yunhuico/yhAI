'use strict';

var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;

var logger = require('../utils/logger');

var Authentication = require('../utils/authentication');

var providerUtil = require('../utils/providerUtil');
var ProviderUtil = new providerUtil("controllerProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;

module.exports = function( app ) {
  app.get('/v1/cmi/usage/total/cpu', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/usage/total/cpu",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.get('/v1/cmi/usage/total/mem', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/usage/total/mem",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.get('/v1/cmi/totaltrend/cpu', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/trend/total/cpu",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.get('/v1/cmi/totaltrend/mem', Authentication.ensureAuthenticated,function(req, res, next) {

    var hostname = req.params.hostname;
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/trend/total/mem",
      method: 'GET',
      json:true,
      headers: {
         'X-Auth-Token': req.session.token
      }
    };
    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.get('/v1/cmi/threshold/total/cpu', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/threshold/total/cpu",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.get('/v1/cmi/threshold/total/mem', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/threshold/total/mem",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.get('/v1/cmi/diskusage', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/diskusage",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });
  app.get('/v1/cmi/alarm/total/cpu', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/alarm/total/cpu",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });
  app.get('/v1/cmi/alarm/total/mem', Authentication.ensureAuthenticated, function( req, res, next ) {
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/alarm/total/mem",
      method: 'GET',
      json: true,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.post('/v1/cluster/:clusterId/registry', Authentication.ensureAuthenticated, function( req, res, next ) {
    
    var clusterId = req.params.clusterId;
    var options = {
      url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.clusterRegistries.replace(/{clusterId}/g, clusterId)),
      method: 'POST',
      json: true,
      body: req.body,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.delete('/v1/cluster/:clusterId/registry', Authentication.ensureAuthenticated, function( req, res, next ) {
    
    var clusterId = req.params.clusterId;
    var options = {
      url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.clusterRegistries.replace(/{clusterId}/g, clusterId)),
      method: 'DELETE',
      json: true,
      body: req.body,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.post('/v1/cluster/:clusterId/pubkey', Authentication.ensureAuthenticated, function( req, res, next ) {
    
    var clusterId = req.params.clusterId;
    var options = {
      url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.clusterPublicKeys.replace(/{clusterId}/g, clusterId)),
      method: 'POST',
      json: true,
      body: req.body,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });

  app.delete('/v1/cluster/:clusterId/pubkey', Authentication.ensureAuthenticated, function( req, res, next ) {
    
    var clusterId = req.params.clusterId;
    
    var options = {
      url: ProviderUtil.rebuildUrl(global.obj.controller_url + urlCfg.controller_api.clusterPublicKeys.replace(/{clusterId}/g, clusterId)),
      method: 'DELETE',
      json: true,
      body: req.body,
      headers: {
        'X-Auth-Token': req.session.token
      }
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
          logger.error('Error retrieving cluster containers data:', error ? error.errno : response.statusCode, body);
          next(new ResponseError(error, response, body));
      }else{
          logger.trace('Retrieved cluster containers data', body);
          res.status(200).send(body);
      }
    };
    request(options, callback);
  });
}
