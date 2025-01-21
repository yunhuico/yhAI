'use strict';

require('sugar');
var path = require('path');
var fs = require('fs');
var request = require('request');

var controllerHA = global.obj.dcosCfg.controllerProvider.ha;
var identityHA = global.obj.dcosCfg.identityProvider.ha;
var controllerProtocol = global.obj.dcosCfg.controllerProvider.auth.protocol;
var indentityProtocol = global.obj.dcosCfg.identityProvider.auth.protocol;

var ProviderUtil = function(providerType) {
    this.providerType = providerType ? providerType : "controllerProvider";

    var caObj = {};
    if(this.providerType === "controllerProvider"){
         if(controllerProtocol === "https"){        	 
         	 var ca = fs.readFileSync(path.join(process.cwd(), global.obj.dcosCfg.controllerProvider.auth.ca));
             caObj = {"ca":ca};
         }
    }else{
         if(indentityProtocol === "https"){
         	 var ca = fs.readFileSync(path.join(process.cwd(), global.obj.dcosCfg.identityProvider.auth.ca))
             caObj = {"ca":ca};
         }
    }

    this.request = request.defaults(caObj);
    
};

ProviderUtil.prototype.rebuildUrl = function(path) {
    var self = this;
    var protocol = "";
    switch (self.providerType){
        case "identityProvider":
          protocol = indentityProtocol;
          break;
        case "controllerProvider":
          protocol = controllerProtocol;
          break;
        default:
          protocol = "http";
    }

    return protocol + "://" + path;
};

ProviderUtil.prototype.parseProviderUrl = function(req, res, next) {
    var self = this;

    if (controllerHA.enabled) {
        global.obj.zkUtil_controller.getProviderUrl();
    } else {
        global.obj.controller_url = controllerHA.controller_url;
    }

    if (identityHA.enabled) {
        global.obj.zkUtil_identity.getProviderUrl();
    } else {
        global.obj.identity_url = identityHA.identity_url;
    }

    return next();
};

module.exports = ProviderUtil;
