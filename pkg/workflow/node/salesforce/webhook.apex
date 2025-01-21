public class WebhookUltrafox implements HttpCalloutMock {

    public static HttpRequest request;
    public static HttpResponse response;

    public HTTPResponse respond(HTTPRequest req) {
        request = req;
        response = new HttpResponse();
        response.setStatusCode(200);
        return response;
    }

    public static String jsonContent(Object triggerNew) {
        String newObjects = '{}';
        if (triggerNew != null) {
            newObjects = JSON.serialize(triggerNew);
        }
        String userId = JSON.serialize(UserInfo.getUserId());
        return newObjects;
    }


    @future(callout=true)
    public static void callout(String url, String content) {

        if (Test.isRunningTest()) {
            Test.setMock(HttpCalloutMock.class, new WebhookUltrafox());
        }

        Http h = new Http();

        HttpRequest req = new HttpRequest();
        req.setEndpoint(url);
        req.setMethod('POST');
        req.setHeader('Content-Type', 'application/json');
        req.setBody(content);

        h.send(req);
    }

}