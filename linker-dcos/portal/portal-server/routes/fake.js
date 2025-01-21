'use strict';

var urlCfg = global.obj.urlCfg;
var linkerConf = global.obj.dcosCfg;
var fs = require('fs');
var fakeCPUPredict = require('../fake/cpu_prediction.json');
var fakeMemPredict = require('../fake/mem_prediction.json');
var fakeTagPredict = require('../fake/tag_prediction.json');

module.exports = function(app) {
	app.get('/fake/cpu', function(req, res, next) {

    res.status(200).send(fakeCPUPredict);
	});

  app.get('/fake/memory', function(req, res, next) {

    res.status(200).send(fakeMemPredict);
  });

  app.get('/fake/tag', function(req, res, next) {

    res.status(200).send(fakeTagPredict);
  });
};
