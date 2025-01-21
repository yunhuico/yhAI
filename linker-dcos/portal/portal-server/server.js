global.obj = {};
var express = require('express');
var cookieParser = require('cookie-parser');
var session = require('express-session');
var bodyParser = require('body-parser');
var path = require('path');
var fs = require('fs');

// var multer  = require('multer');
// var upload = multer();
var compression = require('compression');

// var multipart = require('connect-multiparty');
// var multipartMiddleware = global.obj.multipartMiddleware = multipart();

var config = require('konphyg')(path.resolve(path.dirname(__dirname), 'portal-server/conf'));
var dcosCfg = global.obj.dcosCfg = config('linker-dcos');
var logger = global.obj.logger = require('./utils/logger');
var responseError = require('./utils/responseUtil');
var ZkUtil = require('./utils/zkUtil');

global.obj.urlCfg = config('url');


if (dcosCfg.controllerProvider.ha.enabled) {
    global.obj.zkUtil_controller = new ZkUtil(dcosCfg.controllerProvider.ha.zookeeper_url, "controllerProvider");
};
if (dcosCfg.identityProvider.ha.enabled) {
    global.obj.zkUtil_identity = new ZkUtil(dcosCfg.identityProvider.ha.zookeeper_url, "identityProvider");
}
// if(dcosCfg.controllerProvider.auth.protocol === "https" || dcosCfg.identityProvider.auth.protocol === "https"){
//     global.obj.ca = fs.readFileSync(path.join(__dirname, 'ca.crt'));
// }
var sessionStore;
var app = express();
app.use(compression());
app.use(cookieParser());

if (dcosCfg.ha.enabled) {
    /* use one redis as session storage */
    console.log('HA is enabled, setting up Redis for session store');
    var RedisStore = require('connect-redis')(session);
    sessionStore = new RedisStore(dcosCfg.ha.redis.options);
} else {
    logger.info('Without HA, using MemoryStore');
    var MemoryStore = session.MemoryStore;
    sessionStore = new MemoryStore();
}
app.use(session({
    key: 'JSESSIONID',
    store: sessionStore,
    secret: "aaa",
    resave: false,
    saveUninitialized: false,
    cookie: {
        httpOnly: false,
        maxAge: 21600000
    },
    rolling: true
        // cookie: { secure: true }
}));
//parse post payload
app.use(bodyParser.json({ limit: '20mb' })); // for parsing application/json
app.use(bodyParser.urlencoded({
    extended: true
}));

var staticPath = path.resolve(path.dirname(__dirname), 'portal-ui');
app.use(express.static(staticPath, {
    "index": "login.html"
}));

 // for parsing application/x-www-form-urlencoded
// app.use(bodyParser());
//cross domain

// app.use(function(req, res, next) {
//     res.header('Access-Control-Allow-Credentials', true);
//     res.header('Access-Control-Allow-Origin', '*');
//     res.header('Access-Control-Allow-Methods', 'GET,PUT,POST,DELETE');
//     res.header('Access-Control-Allow-Headers', 'X-Requested-With, X-HTTP-Method-Override, Content-Type, Accept');
//     next();
// });


// dynamically add all the API routes
fs.readdirSync(path.join(__dirname, 'routes')).forEach(function(file) {
    if (file[0] === '.') {
        return;
    }

    require(path.join(__dirname, 'routes', file))(app);
});
/**
 * ResponseError handler
 */
app.use(responseError.handler());

// Add error handler
app.use(function(err, req, res, next) {
    if (!err) {
        return next();
    }
    logger.error('Uncaught error', err, req.path);
    // Send error status code
    res.status(500).send();
});

app.get('/', function(req, res) {
    res.redirect('login.html');
});
if (dcosCfg.http.enabled) {
    app.listen(dcosCfg.http.port_http, function() {
        if (dcosCfg.controllerProvider.ha.enabled) {
            global.obj.zkUtil_controller.connect();
        }
        if (dcosCfg.identityProvider.ha.enabled) {
            global.obj.zkUtil_identity.connect();
        }

        logger.info("linker portal started, listen port " + dcosCfg.http.port_http + ".")
        console.log("linker portal started, listen port " + dcosCfg.http.port_http + ".")

    });
} else {
    var appHttps = require('https');
    var httpsOptions = {
        key: fs.readFileSync(path.join(__dirname, 'ssh-key.pem')),
        cert: fs.readFileSync(path.join(__dirname, 'ssh-cert.pem'))
    };
    appHttps.Server(httpsOptions, app).listen(dcosCfg.http.port_https, function() {
        if (dcosCfg.controllerProvider.ha.enabled) {
            global.obj.zkUtil_controller.connect();
        }
        if (dcosCfg.identityProvider.ha.enabled) {
            global.obj.zkUtil_identity.connect();
        }
        logger.info('linker portal started, listen port ' + dcosCfg.http.port_https + ".");
        console.log('linker portal started, listen port ' + dcosCfg.http.port_https + ".");
    });
}
