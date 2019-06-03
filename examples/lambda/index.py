import json

def handler(event, context):
    print("Received request:", json.dumps(event, indent=4))
    return "CAKE!"
