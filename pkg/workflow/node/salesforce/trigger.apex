trigger {{ .TriggerName }}WebhookTrigger on {{ .SObject }} ( {{ join .Events ", " }} ) {
   String url = '{{ .CallbackUrl }}';
   Object id = Trigger.new[0].get('Id');
   String content = WebhookUltrafox.jsonContent(Trigger.new[0]);
   url = url + '/' + id;
   WebhookUltrafox.callout(url, content);
}