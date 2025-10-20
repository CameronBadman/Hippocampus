import requests
import json

API = "https://z8ba4k81gf.execute-api.ap-southeast-2.amazonaws.com"

payload = {
    "agent_id": "curate_test",
    "text": "My name is Sarah Chen, I'm 34 years old, software engineer at Google. I love pizza and have a cat named Whiskers.",
    "model_id": "amazon.nova-lite-v1:0",
    "importance": "high",
    "timeout_ms": 500
}

print("Testing /agent-curate endpoint...")
print(f"Payload: {json.dumps(payload, indent=2)}\n")

response = requests.post(f"{API}/agent-curate", json=payload)

print(f"Status: {response.status_code}")
print(f"Response: {json.dumps(response.json(), indent=2)}")
