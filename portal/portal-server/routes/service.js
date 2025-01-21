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
    app.get('/appsets', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset + '?sort=-time_create&count=' + req.query.count + '&skip=' + req.query.skip + '&limit=' + req.query.limit+'&skip_group='+req.query.skip_group),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.get('/appsets/:name', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset + '/' + req.params.name),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service detail:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service detail', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.get('/appsets/:name/apps', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset + '/' + req.params.name+'/apps'),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service detail:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service detail', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.post('/appsets', Authentication.ensureAuthenticated, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset),
          method: 'POST',
          json:true,
          body: req.body,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error save appset list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get appset list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/appsets/:name', Authentication.ensureAuthenticated, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset+"/"+req.params.name),
          method: 'PUT',
          json:true,
          body: req.body,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error save appset list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get appset list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/appsets/:name/stop', Authentication.ensureAuthenticated, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset + "/" + req.params.name + "/stop"),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error stop appset:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Stop appset', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/appsets/:name/start', Authentication.ensureAuthenticated, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset + "/" + req.params.name + "/start"),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error start appset:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Start appset', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.delete('/appsets/:name', Authentication.ensureAuthenticated, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.appset + "/" + req.params.name),
          method: 'DELETE',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error delete appset:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Delete appset', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.get('/component', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component + '?name=' + req.query.name+"&appset_name="+req.query.appset_name),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get component detail:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get component detail', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.get('/v1/cmi/server_tag/:hostname', Authentication.ensureAuthenticated,function(req, res, next) {

        var hostname = req.params.hostname;
        var options = {
          url: "http://104.199.206.91:10300/v1/cmi/server_tag/" + hostname,
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });

    app.post('/v1/cmi/nodes/feedback', Authentication.ensureAuthenticated,function(req, res, next) {

        var options = {
          url: "http://104.199.206.91:10300/v1/cmi/nodes/feedback",
          method: 'PUT',
          json: true,
          headers: {
             'X-Auth-Token': req.session.token
          },
          body: req.body
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    }); 

    app.get("/v1/cmi/nodes/tag/:filter"　, Authentication.ensureAuthenticated,function(req, res, next) {
        var filter = req.params["filter"];
        var options = {
          url: "http://104.199.206.91:10300/v1/cmi/nodes?server_tag=" + filter,
          method: 'GET',
          json: true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };

        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });

    app.post("/v1/cmi/nodes/retrain/:filter"　, Authentication.ensureAuthenticated,function(req, res, next) {
        var filter = req.params["filter"];
        var options = {
          url: "http://104.199.206.91:10300/v1/cmi/nodes/retrain",
          method: 'PUT',
          json: true,
          headers: {
             'X-Auth-Token': req.session.token
          },
          body: { filteritem: filter}
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get service list:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get service list', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });

    app.post('/component', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component),
          method: 'POST',
          json:true,
          body: req.body,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var a = JSON.stringify(req.body);
        logger.trace('json', req.body);
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error save component:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Save component', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/component', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component),
          method: 'PUT',
          json:true,
          body: req.body,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var a = JSON.stringify(req.body);
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error save component:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Save component', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/component/scale', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component + "/scale?name=" + req.query.name +"&scaleto="+req.query.scaleto),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error scale component:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Scale component', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/component/start', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component + "/start?name=" + req.query.name),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error start component:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Start component', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.put('/component/stop', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component + "/stop?name=" + req.query.name),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error stop component:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Stop component', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
    });
    app.delete('/component', Authentication.ensureAuthenticated, function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.component + "?name=" + req.query.name ),
          method: 'DELETE',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error delete component:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Delete component', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
  });
  app.get('/container/cadvisor/name', Authentication.ensureAuthenticated,function(req, res, next) {
        var url = urlCfg.dcosclient_api.containerName.replace(/{slaveid}/g, req.query.slaveid).replace(/{taskid}/g, req.query.taskid);
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + url),
          // url: "http://54.238.222.69/cadvisor/"+req.query.slaveid+"/api/linker/dockerid?taskid="+req.query.taskid,
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get container name:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get container name', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
  });
  app.get('/container/read', Authentication.ensureAuthenticated,function(req, res, next) {
    var url = urlCfg.dcosclient_api.sandbox.read.replace(/{slaveid}/g, req.query.slaveid) + req.query.volumePath + "/" + req.query.type + "&offset=" + req.query.offset + "&length=" +req.query.length;
    var options = {
      url: ProviderUtil.rebuildUrl(req.query.clientAddr + url),
      method: 'GET',
      json:true
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
           logger.error('Error get container data:', error ? error.errno : response.statusCode, body);
           next(new ResponseError(error, response, body));
      }else{
           logger.trace('Get container data', body);
           res.status(200).send(body);
      }
    };
    request(options, callback);
  });
  app.get('/container/browse', Authentication.ensureAuthenticated,function(req, res, next) {
    var url = urlCfg.dcosclient_api.sandbox.browse.replace(/{slaveid}/g, req.query.slaveid) + req.query.path;
    var options = {
      url: ProviderUtil.rebuildUrl(req.query.clientAddr + url),
      method: 'GET',
      json:true
    };

    var callback = function(error, response, body) {
      if(error || response.statusCode >= 400){
           logger.error('Error browse container folder:', error ? error.errno : response.statusCode, body);
           next(new ResponseError(error, response, body));
      }else{
           logger.trace('Get container folder', body);
           res.status(200).send(body);
      }
    };
    request(options, callback);
  });
  app.get('/container/:id', Authentication.ensureAuthenticated,function(req, res, next) {
        var url = urlCfg.dcosclient_api.containerInfo.replace(/{slaveid}/g, req.query.slaveid).replace(/{taskid}/g, req.params.id);
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + url),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error get container detail:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Get container detail', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
  });
   app.put('/container/:id/redeploy', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.container + '/' + req.params.id+"/redeploy"),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error redeploy container:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Redeploy container', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
  });
  app.put('/container/:id/kill', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.container + '/' + req.params.id+"/kill"),
          method: 'PUT',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error kill container:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Kill container', body);
               res.status(200).send(body);
          }
        };
        request(options, callback);
  });
  app.get('/webconsole',Authentication.ensureAuthenticated,function(req,res,next){
  		var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.webconsole + '?cid='+req.query.cid),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
       };
        var callback = function(error, response, body) {
          if(error || response.statusCode >= 400){
               logger.error('Error connect console:', error ? error.errno : response.statusCode, body);
               next(new ResponseError(error, response, body));
          }else{
               logger.trace('Connect console', body);
               body=body.replace(/href="/g,"$&http://"+req.query.clientAddr+urlCfg.dcosclient_api.webconsole+'/');
               body=body.replace(/src="./g,"src=\"http://"+req.query.clientAddr+urlCfg.dcosclient_api.webconsole);
               body=body.replace(/<script/,"<script>window.jump={};window.jump.host='"+req.query.clientAddr+":10022';"+"window.jump.pathname='/console/';</script>\n$&");
               res.status(200).send(body);
          }
        };
        request(options, callback);
  })



};
