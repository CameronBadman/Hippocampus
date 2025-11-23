#!/bin/bash
# test-lambda-safety.sh
# Demo: Store memory and then test safety agent

API_URL="https://rbf04f5hud.execute-api.ap-southeast-2.amazonaws.com"
AGENT_ID="safety_demo_parent"

# -----------------------------
# Step 1: Insert memory
# -----------------------------
INSERT_KEY="emma_allergy"
INSERT_TEXT="My daughter Emma is 5 years old and has a severe shellfish allergy. Even small amounts can cause anaphylaxis."

insert_payload=$(cat <<EOF
{
  "agent_id": "$AGENT_ID",
  "key": "$INSERT_KEY",
  "text": "$INSERT_TEXT"
}
EOF
)

echo "Inserting memory..."
insert_response=$(curl -s -X POST "$API_URL/insert" \
  -H "Content-Type: application/json" \
  -d "$insert_payload")

echo "Insert response:"
echo "$insert_response"
echo
echo "-----------------------------------"
echo

# -----------------------------
# Step 2: Query safety agent
# -----------------------------
USER_MESSAGE="I'm at the grocery store. I'm thinking of buying some shrimp to cook for Emma tonight. She's never tried it before!"

safety_payload=$(cat <<EOF
{
  "agent_id": "$AGENT_ID",
  "message": "$USER_MESSAGE"
}
EOF
)

echo "Querying safety agent..."
safety_response=$(curl -s -X POST "$API_URL/agent-safety" \
  -H "Content-Type: application/json" \
  -d "$safety_payload")

echo "Safety agent response:"
echo "$safety_response"
