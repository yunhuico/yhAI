'use strict';

require('sugar');
var logger = global.obj.logger;

module.exports.ensureAuthenticated = function (req, res, next) {
    if(!req.session || !req.session.token){
    	var error={"errormsg":"Have not signin.","code":"Web.NotSignIn"};
        res.status(401).send(error);
    }else{
    	return next();
    }
};
