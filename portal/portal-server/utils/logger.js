require('sugar');
var winston = require('winston'),path = require('path');
var LogConf = (global.obj) ? global.obj.dcosCfg : {
  	logging: {
    	console: {
      		enabled: false
    	},
   	 	file: {
      		enabled: true
    	}
  	}
};
 
var LogLevels = {
  	levels: {
    	error: 0,
    	warn: 1,
    	info: 2,
    	debug: 3,
   	 	trace: 4
  	},
  	colors: {
    	error: 'red',
    	warn: 'orange',
    	info: 'green',
    	debug: 'blue',
    	trace: 'yellow'
  	}
};

var transports = [];
if (LogConf.logging.console.enabled) {
  	transports.push(new(winston.transports.Console)({
    	level: LogConf.logging.console.level ? LogConf.logging.console.level.toLowerCase() : 'info',
    	json: true,
    	handleExceptions: true,
    	colorize: true,
    	timestamp: function () {
      		return Date.create().format('{dd} {Mon} {yyyy} {hh}:{mm}:{ss},{fff}');
    	}
  	}));
}
if (LogConf.logging.file.enabled) {
    var logFileName = "";
    if(LogConf.purpose.production){
        var dir = LogConf.logging.file.dir || "/var/log/";
        logFileName = dir + 'linker-dcos-portal.log';
    }else{
        logFileName = path.resolve(path.dirname(__dirname),'logs', 'linker-dcos-portal.log');
    }
  	transports.push(new(winston.transports.File)({
    	level: LogConf.logging.file.level ? LogConf.logging.file.level.toLowerCase() : 'info',
   	 	filename: logFileName,
    	handleExceptions: true,
    	maxsize: LogConf.logging.file.maxSizeMB * 1024 * 1024 || 10485760, //10M
    	maxFile: LogConf.logging.file.maxFile || 10,
    	timestamp: function () {
      		return Date.create().format('{dd} {Mon} {yyyy} {hh}:{mm}:{ss},{fff}');
    	},
    	json: true
  	}));
}
var logger = new(winston.Logger)({
  levels: LogLevels.levels,
  transports: transports,
  exitOnError: false
});
winston.addColors(LogLevels.colors);
module.exports = logger;
