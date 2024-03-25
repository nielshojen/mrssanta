import functions_framework
import json

request = {
  "rules_received" : None,
  "rules_processed" : None,
}

@functions_framework.http
def xsrf(request):
    request_json = request.get_json(silent=True)
    request_args = request.args
    
    response = request_json

    if request_args and "machine_id" in request_args:
        response['ID'] = request_args['machine_id']

    print(response)

    return {"status": "ok"}, 200, {'Content-Type': 'application/json'}