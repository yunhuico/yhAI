'use strict';

var logger = global.obj.logger;

/**
 * Default error if the response is not well formed.
 */
var defaultError = {
  'code': 'Web.ServiceUnavailable',
  'errormsg' : 'Web.ServiceUnavailable'
};
var specificError = {
  'code': '',
  'errormsg' : ''
};

/**
 * Wraps the result of an error from a call to a Provider.
 *
 * If the response is not well formed, returns a generic
 * response that the service is unavailable.
 * If the response is well formed then it returns the response statusCode and body
 * minus stackTrace and description parameters (for security reasons);
 */
function ResponseError(error, response, body) {
  this.error = error;
  this.response = response;
  this.body = body;
  if (error) {
    this.statusCode = 503;
    this.responseBody = defaultError;
  } else if (this.response && this.response.statusCode && this.body && this.body.error) {
    this.statusCode = this.response.statusCode;
    this.responseBody = this.body.error;
  } else {
    this.statusCode = this.response.statusCode;
    specificError = {'code': this.response.statusCode,'errormsg' : this.response.statusMessage, 'data': this.body};
    this.responseBody = specificError;
  }
}

ResponseError.prototype.getDetails = function () {
  return {
    statusCode: this.statusCode,
    responseBody: this.responseBody
  };
};

function handler() {
  return function (err, req, res, next) {
    if (err && err.constructor === ResponseError) {
      logger.error('Handling a response error', err.getDetails());
      // res.send(err.statusCode, err.responseBody);
      res.status(err.statusCode).send(err.responseBody);
    } else {
      next(err);
    }
  };
}


module.exports = {
  model: ResponseError,
  handler: handler
};
