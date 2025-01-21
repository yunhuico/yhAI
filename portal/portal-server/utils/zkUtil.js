'use strict';
require('sugar');
var zookeeper = require('node-zookeeper-client');

var dcosCfg = global.obj.dcosCfg;

var logger = global.obj.logger;

var ZkUtil = function(zookeeper_url, providerType) {
      this.url = zookeeper_url;
      this.client = zookeeper.createClient(this.url);
      this.controllerEndpoints = []; 
      this.providerType = providerType;
      if(providerType == "controllerProvider"){
          this.rootPath = "/controller";
      }else if(providerType == "identityProvider"){
          this.rootPath = "/userMgmt";
      }
      
};

ZkUtil.prototype.getClient = function() {
  if (this.client != null) {
    return this.client;
  } else {
    this.client = zookeeper.createClient(this.url);
    return this.client;
  }
};

ZkUtil.prototype.connect = function() {
  var self = this;
  if (this.client != null) {
    this.client.once('connected', function() {
      self.watchProvider();
    }).connect();
  }
};

ZkUtil.prototype.closeConnection = function() {
  if (this.client != null) {
    this.client.close();
  }
};

ZkUtil.prototype.setProviderEndpoints = function(children) {
  var self = this;
  if(self.providerType == "controllerProvider"){
     global.obj.controller_urls = [];
  }else if(self.providerType == "identityProvider"){
     global.obj.identity_urls = [];
  }
  var childrenLen = children.length;
  if (childrenLen > 0) {
    children.forEach(function(child) {
      self.client.getData(self.rootPath + '/' + child, function(error, data, stat) {
        if (error) {
          // console.log(error.stack);
          logger.error('Get zookeeper set children error:', error.stack);
          return;
        }
        if(self.providerType == "controllerProvider"){
           global.obj.controller_urls.push(data.toString('utf8'));
        }else if(self.providerType == "identityProvider"){
           global.obj.identity_urls.push(data.toString('utf8'));
        }
        
      });
    });
  } else {
    // console.log('Can not connect to provider from zookeeper');
    logger.error('Can not connect to provider from zookeeper');
    return;
  }
};

ZkUtil.prototype.getProviderUrl = function(req, res, next) {
      var self = this;
      var childrenLen = 0;
      if(self.providerType == "controllerProvider"){
          childrenLen = global.obj.controller_urls.length;
          global.obj.controller_url = global.obj.controller_urls[Math.floor(Math.random() * childrenLen)];   
      }else if(self.providerType == "identityProvider"){
          childrenLen = global.obj.identity_urls.length;
          global.obj.identity_url = global.obj.identity_urls[Math.floor(Math.random() * childrenLen)];   
      }
    
  // console.log('Got data: %s', global.app.controller_url);
  // next();
};

ZkUtil.prototype.watchProvider = function() {
  var self = this;
  this.client.getChildren(self.rootPath, function(event) {
    console.log('Got event: %s.', event);
    self.watchProvider()
  }, function(error, children, stats) {
    if (error) {
      // console.log(error.stack);
      logger.error('Get zookeeper get children error:', error.stack);
      return;
    }
    console.log(self.providerType + ' children are: %j.', children);
    logger.trace(self.providerType + ' children are: %j.', children);
    self.setProviderEndpoints(children);
  });
};

module.exports = ZkUtil;