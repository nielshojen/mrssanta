import functions_framework
import json

request = {
  "rules_received" : None,
  "rules_processed" : None,
}

@functions_framework.http
def postflight(request):
    request_json = request.get_json(silent=True)
    request_args = request.args
    
    response = {}

    if request_args and "machine_id" in request_args:
        response['ID'] = request_args.getlist('machine_id')[0]

    if request_json and "rules_received" in request_json:
        response['rules_received'] = request_json['rules_received']

    if request_json and "rules_processed" in request_json:
        response['rules_processed'] = request_json['rules_processed']

    print(response)

    return {"status": "ok"}, 200, {'Content-Type': 'application/json'}