'use strict';

require('sugar');
var logger = global.obj.logger;
var dcosCfg = global.obj.dcosCfg;
var protocol = dcosCfg.http.enabled ? "http://" : "https://";
var port = dcosCfg.http.enabled ? dcosCfg.http.port_http : dcosCfg.http.port_https;
module.exports = {
	base64Encode : function(params){
        return new Buffer(params).toString("base64"); 
	},
    base64Decode : function(params){
        return new Buffer(params, 'base64').toString();
    }
};
