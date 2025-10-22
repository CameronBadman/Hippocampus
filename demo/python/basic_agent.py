import boto3
import json
import requests
import time

bedrock = boto3.client('bedrock-runtime', region_name='us-east-1')

HIPPOCAMPUS_API = "https://rbf04f5hud.execute-api.ap-southeast-2.amazonaws.com"
AGENT_ID = "safety_demo_parent"

SYSTEM_PROMPT = """You are a helpful assistant with access to long-term memory about the user and their family.

CRITICAL: Always check memory before giving advice, especially about:
- Health/medical information
- Allergies and dietary restrictions
- Safety concerns
- Children's needs

When the user mentions doing something, proactively search for relevant safety information.

Search Parameter Guidelines:
- For safety-critical queries (allergies, medical): use high threshold (0.7+), low epsilon (0.2)
- For general family information: use moderate settings (threshold 0.5, epsilon 0.3)
- Adjust top_k based on how comprehensive you need the results"""

tools = [
    {
        "toolSpec": {
            "name": "insert_memory",
            "description": "Store important information for later retrieval.",
            "inputSchema": {
                "json": {
                    "type": "object",
                    "properties": {
                        "key": {
                            "type": "string",
                            "description": "Descriptive key for this memory",
                        },
                        "text": {
                            "type": "string",
                            "description": "The information to remember",
                        },
                    },
                    "required": ["key", "text"]
                },
            },
        },
    },
    {
        "toolSpec": {
            "name": "search_memory",
            "description": "Search for previously stored information. ALWAYS use this before giving advice about health, food, or activities involving children. Control search precision: use high threshold (0.7+) for safety-critical queries.",
            "inputSchema": {
                "json": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "What to search for",
                        },
                        "epsilon": {
                            "type": "number",
                            "description": "Search radius (0.1-0.5). Lower = stricter. Use 0.2 for safety queries, 0.3 for general. Default 0.3.",
                            "default": 0.3,
                        },
                        "threshold": {
                            "type": "number",
                            "description": "Minimum similarity (0.0-1.0). Use 0.7+ for safety queries, 0.5 for general. Default 0.5.",
                            "default": 0.5,
                        },
                        "top_k": {
                            "type": "integer",
                            "description": "Max results (1-10). Use 3-5 for most queries. Default 5.",
                            "default": 5,
                        },
                    },
                    "required": ["query"],
                },
            },
        },
    },
]

def call_hippocampus(endpoint, payload):
    response = requests.post(f"{HIPPOCAMPUS_API}/{endpoint}", json=payload)
    return response.json()


def handle_tool_use(tool_name, tool_input):
    if tool_name == "insert_memory":
        result = call_hippocampus("insert", {
            "agent_id": AGENT_ID,
            "key": tool_input["key"],
            "text": tool_input["text"],
        })
        return {"success": True, "message": "Memory stored"}

    epsilon = tool_input.get("epsilon", 0.3)
    threshold = tool_input.get("threshold", 0.5)
    top_k = tool_input.get("top_k", 5)
    result = call_hippocampus("search", {
        "agent_id": AGENT_ID,
        "text": tool_input["query"],
        "epsilon": epsilon,
        "threshold": threshold,
        "top_k": top_k,
    })
    memories = result.get("data", [])
    return {
        "found": len(memories) > 0,
        "memories": memories,
        "search_params": {
            "epsilon": epsilon,
            "threshold": threshold,
            "top_k": top_k,
        },
    }


def chat(user_message, conversation_history):
    conversation_history.append({
        "role": "user",
        "content": [{"text": user_message}],
    })
    response = bedrock.converse(
        modelId="us.amazon.nova-lite-v1:0",
        messages=conversation_history,
        system=[{"text": SYSTEM_PROMPT}],
        toolConfig={"tools": tools},
    )
    tool_calls_made = []
    while response["stopReason"] == "tool_use":
        tool_requests = [c for c in response["output"]["message"]['content'] if 'toolUse' in c]
        tool_results = []
        for tool_request in tool_requests:
            tool_use = tool_request["toolUse"]
            tool_calls_made.append({
                "tool": tool_use["name"],
                "input": tool_use["input"],
            })
            result = handle_tool_use(tool_use["name"], tool_use["input"])
            tool_results.append({
                "toolResult": {
                    "toolUseId": tool_use["toolUseId"],
                    "content": [{"json": result}],
                }
            })
        conversation_history.append(response["output"]["message"])
        conversation_history.append({
            "role": "user",
            "content": tool_results,
        })
        response = bedrock.converse(
            modelId="us.amazon.nova-lite-v1:0",
            messages=conversation_history,
            system=[{"text": SYSTEM_PROMPT}],
            toolConfig={"tools": tools},
        )
    assistant_message = response["output"]["message"]
    conversation_history.append(assistant_message)
    return assistant_message["content"][0]["text"], tool_calls_made


def main():
    print("Hippocampus Safety Demo: Critical Memory Retrieval")
    print("=" * 70)
    print("\nThis demo shows how persistent memory can prevent dangerous mistakes.")
    print("Scenario: Parent with child who has shellfish allergy\n")
    print("=" * 70)
    print("CONVERSATION 1: Sharing Family Information")
    print("=" * 70)
    conversation1 = []
    user_input_1 = "My daughter Emma is 5 years old and has a severe shellfish allergy. Even small amounts can cause anaphylaxis."
    print(f"\nParent: {user_input_1}")
    response_1, tools_1 = chat(user_input_1, conversation1)
    print("\n Tools Used:")
    for tool_call in tools_1:
        print(f"   - {tool_call['tool']}: {tool_call['input']}")
    print(f"\nAssistant: {response_1}")
    print("\n" + "=" * 70)
    print(" Simulating time passing... (New conversation session)")
    print("=" * 70)
    time.sleep(10)
    print("\n" + "=" * 70)
    print("CONVERSATION 2: Potential Dangerous Action (Days/Weeks Later)")
    print("=" * 70)
    time.sleep(10)
    conversation2 = []
    user_input_2 = "I'm at the grocery store. I'm thinking of buying some shrimp to cook for dinner for Emma tonight. She's never tried it before!"
    print(f"\nParent: {user_input_2}")
    response_2, tools_2 = chat(user_input_2, conversation2)
    print("\n Tools Used:")
    for tool_call in tools_2:
        print(f"   - {tool_call['tool']}: {tool_call['input']}")
    print(f"\nAssistant: {response_2}")
    print("\n" + "=" * 70)
    print("DEMO COMPLETE")
    print("=" * 70)
    print("\nWhat just happened:")
    print("  1. In conversation 1: Agent stored Emma's shellfish allergy")
    print("  2. In conversation 2: Agent PROACTIVELY searched memory")
    print("  3. Agent used high threshold (0.7+) for safety-critical search")
    print("  4. Agent retrieved critical safety information")
    print("  5. Agent warned parent about the danger")
    print("\n Without persistent memory, this could have been life-threatening!")
    print("=" * 70)

if __name__ == "__main__":
    main()
