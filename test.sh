API="https://jpdbd7nyd7.execute-api.us-east-1.amazonaws.com"

# Clear old S3 data for fresh start
aws s3 rm s3://hippocampus-12345/agents/ --recursive

# Test 1: Insert a memory
curl -X POST $API/insert \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent_prod",
    "key": "favorite color preference",
    "text": "User prefers blue and purple colors"
  }'

# Test 2: Search for it
curl -X POST $API/search \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent_prod",
    "text": "color preferences",
    "epsilon": 0.3
  }'

# Test 3: Insert another memory
curl -X POST $API/insert \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent_prod",
    "key": "dietary restrictions",
    "text": "User is allergic to peanuts and shellfish"
  }'

# Test 4: Search again
curl -X POST $API/search \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent_prod",
    "text": "food allergies",
    "epsilon": 0.3
  }'
