'use strict';

var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;

var logger = require('../utils/logger');

var Authentication = require('../utils/authentication');

var providerUtil = require('../utils/providerUtil');
var ProviderUtil = new providerUtil("controllerProvider");
var request = ProviderUtil.request;

var ResponseError = require('../utils/responseUtil').model;

module.exports = function(app) {
	app.get('/monitor/slavenodes', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + '/mesos/state-summary'),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
		    if(error || response.statusCode >= 400){
		        logger.error('Error retrieving cluster nodes data:', error ? error.errno : response.statusCode, body);
		        next(new ResponseError(error, response, body));
		    }else{
		        logger.trace('Retrieved cluster nodes data', body);
		        res.status(200).send(body);
		    }
		};
        request(options, callback);
    });

    app.get('/v1/cmi/trend/:hostname/cpu', Authentication.ensureAuthenticated,function(req, res, next) {

    	var hostname = req.params.hostname;
	    var options = {
	      url: "http://104.199.206.91:10300/v1/cmi/trend/" + hostname + "/cpu",
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

    app.get('/v1/cmi/trend/:hostname/mem', Authentication.ensureAuthenticated,function(req, res, next) {

    	var hostname = req.params.hostname;
	    var options = {
	      url: "http://104.199.206.91:10300/v1/cmi/trend/" + hostname + "/mem",
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

	app.get('/v1/cmi/usage/:hostname/cpu', Authentication.ensureAuthenticated,function(req, res, next) {

  	var hostname = req.params.hostname;
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/usage/" + hostname + "/cpu",
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

	app.get('/v1/cmi/usage/:hostname/mem', Authentication.ensureAuthenticated,function(req, res, next) {

  	var hostname = req.params.hostname;
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/usage/" + hostname + "/mem",
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

	app.get('/v1/cmi/threshold/:hostname/cpu', Authentication.ensureAuthenticated,function(req, res, next) {

  	var hostname = req.params.hostname;
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/threshold/" + hostname + "/cpu",
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

	app.get('/v1/cmi/threshold/:hostname/mem', Authentication.ensureAuthenticated,function(req, res, next) {

  	var hostname = req.params.hostname;
    var options = {
      url: "http://104.199.206.91:10300/v1/cmi/threshold/" + hostname + "/mem",
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

	app.get('/monitor/containers', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.monitor + '/containers'),
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

	app.get('/monitor/service/containers', Authentication.ensureAuthenticated,function(req, res, next) {
        var options = {
          url: ProviderUtil.rebuildUrl(req.query.clientAddr + urlCfg.dcosclient_api.monitor + '/' + req.query.groupid + '/containers'),
          method: 'GET',
          json:true,
          headers: {
             'X-Auth-Token': req.session.token
          }
        };
        var callback = function(error, response, body) {
		     if(error || response.statusCode >= 400){
		          logger.error('Error retrieving service containers data:', error ? error.errno : response.statusCode, body);
		          next(new ResponseError(error, response, body));
		        }else{
		          logger.trace('Retrieved service containers data', body);
		          res.status(200).send(body);
		         }
		     };
        request(options, callback);
    });

	app.get('/nodemonitoring', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res,next) {
	      	var ip = req.query.ip;
	      	var slaveid = req.query.slaveid;
	      	var url = urlCfg.cadvisor.nodeMonitoring.replace(/{ip}/g, ip).replace(/{slaveid}/g, slaveid);
	      	var options = {
	        		url: url,
	        		method: 'GET',
	        		json:true,
	        		headers: {
	            		'X-Auth-Token': req.session.token
	        		}
	      	};
	      	var callback = function(error, response, body) {
		      	if(error || response.statusCode >= 400){
		          logger.error('Error retrieving node monitoring data:', error ? error.errno : response.statusCode, body);
		          next(new ResponseError(error, response, body));
		        }else{
		          logger.trace('Retrieved node monitoring data', body);
		          res.status(200).send(body);
		         }
		     };
		    logger.trace("Start to get node monitoring data by request " + options.url);
		    request(options, callback);
	});

	app.get('/nodespec', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res,next) {
	      	var ip = req.query.ip;
	      	var slaveid = req.query.slaveid;
	      	var url = urlCfg.cadvisor.nodeSpec.replace(/{ip}/g, ip).replace(/{slaveid}/g, slaveid);
	      	var options = {
	        		url: url,
	        		method: 'GET',
	        		json:true,
	        		headers: {
	            		'X-Auth-Token': req.session.token
	        		}
	      	};
	      	var callback = function(error, response, body) {
		      	if(error || response.statusCode >= 400){
		          logger.error('Error retrieving node spec data:', error ? error.errno : response.statusCode, body);
		          next(new ResponseError(error, response, body));
		        }else{
		          logger.trace('Retrieved node spec data', body);
		          res.status(200).send(body);
		         }
		     };
		    logger.trace("Start to get node spec data by request " + options.url);
		    request(options, callback);
	});

	app.get('/containermonitoring', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res,next) {
	      	var ip = req.query.ip;
	      	var slaveid = req.query.slaveid;
	      	var dockername = req.query.dockername;
	      	var url = urlCfg.cadvisor.containerMonitoring.replace(/{ip}/g, ip).replace(/{slaveid}/g, slaveid).replace(/{dockername}/g, dockername);
	      	var options = {
	        		url: url,
	        		method: 'GET',
	        		json:true,
	        		headers: {
	            		'X-Auth-Token': req.session.token
	        		}
	      	};
	      	var callback = function(error, response, body) {
		      	if(error || response.statusCode >= 400){
		          logger.error('Error retrieving docker monitoring data:', error ? error.errno : response.statusCode, body);
		          next(new ResponseError(error, response, body));
		        }else{
		          logger.trace('Retrieved docker monitoring data', body);
		          res.status(200).send(body);
		         }
		     };
		    logger.trace("Start to get docker monitoring data by request " + options.url);
		    request(options, callback);
	});

	app.get('/containerspec', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res,next) {
	      	var ip = req.query.ip;
	      	var slaveid = req.query.slaveid;
	      	var dockername = req.query.dockername;
	      	var url = urlCfg.cadvisor.containerSpec.replace(/{ip}/g, ip).replace(/{slaveid}/g, slaveid).replace(/{dockername}/g, dockername);
	      	var options = {
	        		url: url,
	        		method: 'GET',
	        		json:true,
	        		headers: {
	            		'X-Auth-Token': req.session.token
	        		}
	      	};
	      	var callback = function(error, response, body) {
		      	if(error || response.statusCode >= 400){
		          logger.error('Error retrieving docker spec data:', error ? error.errno : response.statusCode, body);
		          next(new ResponseError(error, response, body));
		        }else{
		          logger.trace('Retrieved docker spec data', body);
		          res.status(200).send(body);
		         }
		     };
		    logger.trace("Start to get docker spec data by request " + options.url);
		    request(options, callback);
	});

	app.get('/machineinfo', Authentication.ensureAuthenticated, ProviderUtil.parseProviderUrl, function(req, res,next) {
	      	var ip = req.query.ip;
	      	var slaveid = req.query.slaveid;
	      	var url = urlCfg.cadvisor.machineinfo.replace(/{ip}/g, ip).replace(/{slaveid}/g, slaveid);
	      	var options = {
	        		url: url,
	        		method: 'GET',
	        		json:true,
	        		headers: {
	            		'X-Auth-Token': req.session.token
	        		}
	      	};
	      	var callback = function(error, response, body) {
		      	if(error || response.statusCode >= 400){
		          logger.error('Error retrieving machine info:', error ? error.errno : response.statusCode, body);
		          next(new ResponseError(error, response, body));
		        }else{
		          logger.trace('Retrieved machine info', body);
		          res.status(200).send(body);
		         }
		     };
		    logger.trace("Start to get machine info by request " + options.url);
		    request(options, callback);
	});
};
