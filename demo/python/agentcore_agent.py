import boto3
import json
import requests
from typing import List, Dict

bedrock = boto3.client('bedrock-runtime', region_name='us-east-1')

HIPPOCAMPUS_API = ""
AGENT_ID = "agentcore_demo"

SYSTEM_PROMPT = """You are an intelligent memory management agent with two distinct modes:

1. REASONING MODE: When the user shares information, you analyze it deeply to:
   - Identify multiple distinct facts or topics
   - Generate specific, searchable key names for each fact
   - Decide which information is worth storing
   - Create multiple targeted memories instead of one large memory

2. RESPONSE MODE: After storing memories, respond naturally to the user.

Key Generation Guidelines:
- Use descriptive, searchable keys like "favorite_food_pizza" not just "food"
- Create separate memories for separate facts
- Use consistent naming: category_subcategory_detail
- Examples: "travel_history_italy_2024", "dietary_restriction_peanut_allergy", "work_preference_remote"

Search Parameter Guidelines:
You have full control over search precision. Adapt based on the query:

- Safety-critical queries (allergies, medical):
  epsilon=0.15, threshold=0.7, top_k=3
  
- Exact lookups (specific facts):
  epsilon=0.175, threshold=0.6, top_k=1-3
  
- General queries (exploring related info):
  epsilon=0.2, threshold=0.5, top_k=5
  
- Broad discovery (finding anything relevant):
  epsilon=0.4, threshold=0.4, top_k=10

Use your judgment. If a search returns nothing, you can retry with relaxed parameters.

Always explain what you're storing and why."""

tools = [
    {
        "toolSpec": {
            "name": "insert_memory",
            "description": "Store a single memory with a descriptive key. Call this multiple times for multiple facts.",
            "inputSchema": {
                "json": {
                    "type": "object",
                    "properties": {
                        "key": {
                            "type": "string",
                            "description": "Specific, searchable key using format: category_subcategory_detail"
                        },
                        "text": {
                            "type": "string",
                            "description": "The actual information to remember"
                        },
                        "reasoning": {
                            "type": "string",
                            "description": "Brief explanation of why this key was chosen"
                        }
                    },
                    "required": ["key", "text", "reasoning"]
                }
            }
        }
    },
    {
        "toolSpec": {
            "name": "search_memory",
            "description": "Search for previously stored information using semantic similarity. You control search precision with epsilon, threshold, and top_k parameters.",
            "inputSchema": {
                "json": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "What to search for"
                        },
                        "reasoning": {
                            "type": "string",
                            "description": "Why you're searching for this"
                        },
                        "epsilon": {
                            "type": "number",
                            "description": "Search radius (0.1-0.5). Lower = stricter matching. Use 0.2 for exact matches, 0.4 for broad searches.",
                            "default": 0.3
                        },
                        "threshold": {
                            "type": "number",
                            "description": "Minimum similarity score (0.0-1.0). Higher = stricter. Use 0.7+ for safety-critical, 0.4-0.6 for general searches.",
                            "default": 0.5
                        },
                        "top_k": {
                            "type": "integer",
                            "description": "Maximum number of results (1-10). Use 1-3 for precise answers, 5-10 for comprehensive searches.",
                            "default": 5
                        }
                    },
                    "required": ["query", "reasoning"]
                }
            }
        }
    }
]

def call_hippocampus(endpoint: str, payload: Dict) -> Dict:
    response = requests.post(f"{HIPPOCAMPUS_API}/{endpoint}", json=payload)
    return response.json()

def handle_tool_use(tool_name: str, tool_input: Dict) -> Dict:
    if tool_name == "insert_memory":
        result = call_hippocampus("insert", {
            "agent_id": AGENT_ID,
            "key": tool_input["key"],
            "text": tool_input["text"]
        })
        return {
            "success": True,
            "key": tool_input["key"],
            "reasoning": tool_input.get("reasoning", ""),
            "message": "Memory stored successfully"
        }
    
    elif tool_name == "search_memory":
        epsilon = tool_input.get("epsilon", 0.3)
        threshold = tool_input.get("threshold", 0.5)
        top_k = tool_input.get("top_k", 5)
        
        result = call_hippocampus("search", {
            "agent_id": AGENT_ID,
            "text": tool_input["query"],
            "epsilon": epsilon,
            "threshold": threshold,
            "top_k": top_k
        })
        
        memories = result.get("data", [])
        
        return {
            "found": len(memories) > 0,
            "count": len(memories),
            "memories": memories,
            "reasoning": tool_input.get("reasoning", ""),
            "search_params": {
                "epsilon": epsilon,
                "threshold": threshold,
                "top_k": top_k
            }
        }
    
    return {"error": "Unknown tool"}

def chat(user_message: str, conversation_history: List[Dict]) -> str:
    conversation_history.append({
        "role": "user",
        "content": [{"text": user_message}]
    })
    
    response = bedrock.converse(
        modelId="us.amazon.nova-lite-v1:0",
        messages=conversation_history,
        system=[{"text": SYSTEM_PROMPT}],
        toolConfig={"tools": tools}
    )
    
    tool_use_count = 0
    
    while response['stopReason'] == 'tool_use':
        tool_requests = [c for c in response['output']['message']['content'] if 'toolUse' in c]
        
        tool_results = []
        for tool_request in tool_requests:
            tool_use = tool_request['toolUse']
            tool_use_count += 1
            
            print(f"\n Tool Call #{tool_use_count}: {tool_use['name']}")
            print(f"     Input: {json.dumps(tool_use['input'], indent=6)}")
            
            result = handle_tool_use(tool_use['name'], tool_use['input'])
            
            print(f"     Result: {json.dumps(result, indent=6)}")
            
            tool_results.append({
                "toolResult": {
                    "toolUseId": tool_use['toolUseId'],
                    "content": [{"json": result}]
                }
            })
        
        conversation_history.append(response['output']['message'])
        conversation_history.append({
            "role": "user",
            "content": tool_results
        })
        
        response = bedrock.converse(
            modelId="us.amazon.nova-lite-v1:0",
            messages=conversation_history,
            system=[{"text": SYSTEM_PROMPT}],
            toolConfig={"tools": tools}
        )
    
    assistant_message = response['output']['message']
    conversation_history.append(assistant_message)
    
    return assistant_message['content'][0]['text']


def demo_scenario():
    print(" Hippocampus AgentCore Demo")
    print("=" * 60)
    print("\nThis demo shows intelligent memory decomposition.")
    print("The agent will analyze input and create multiple targeted memories.\n")
    
    conversation = []
    
    scenarios = [
        {
            "description": "Dense personal profile with 15+ extractable facts",
            "input": "My name is Sarah Chen, I'm 34 years old, software engineer at Google working on cloud infrastructure. I live in Seattle with my husband Michael who's a high school math teacher. We have two kids - Emma is 5 and has a severe peanut allergy, Jake is 8 and plays competitive soccer. I'm training for the Seattle marathon in November, currently running 40 miles a week. I'm vegetarian but eat fish occasionally, allergic to shellfish and latex. I went to MIT for undergrad, got my masters at Stanford. My parents live in San Francisco, I try to visit them monthly. I love sci-fi novels, currently reading the Three Body Problem series. I speak Mandarin fluently, learning Spanish with the kids. We have a golden retriever named Cosmo who's 3 years old. I'm trying to quit coffee, down to one cup in the morning. My favorite programming language is Rust, but I work mostly in Go and Python at Google. I play piano, been taking lessons for 2 years now."
        },
        {
            "description": "Targeted query that should retrieve subset",
            "input": "What programming languages do I use?"
        },
        {
            "description": "Different targeted query",
            "input": "Tell me about my family"
        },
        {
            "description": "Safety-critical query",
            "input": "Can my kids eat peanut butter sandwiches?"
        }
    ]
    
    for i, scenario in enumerate(scenarios, 1):
        print(f"\n{'='*60}")
        print(f"Scenario {i}: {scenario['description']}")
        print(f"{'='*60}")
        print(f"\nUser: {scenario['input']}")
        
        response = chat(scenario['input'], conversation)
        
        print(f"\nAgent: {response}")
        
        if i < len(scenarios):
            input("\n[Press Enter to continue to next scenario...]")
    
    print("\n" + "="*60)
    print("Demo complete! Notice how the agent:")
    print("  1. Decomposed one input into multiple memories")
    print("  2. Used descriptive, searchable keys")
    print("  3. Retrieved relevant memories when needed")
    print("  4. Adapted search parameters based on query type")
    print("="*60)

def interactive_mode():
    print("Hippocampus AgentCore - Interactive Mode")
    print("=" * 60)
    print("The agent will intelligently manage memories.")
    print("Type 'quit' to exit.\n")
    
    conversation = []
    
    while True:
        user_input = input("\nYou: ")
        if user_input.lower() in ['quit', 'exit']:
            break
        
        response = chat(user_input, conversation)
        print(f"\nAgent: {response}")

def main():
    import sys
    
    if len(sys.argv) > 1 and sys.argv[1] == '--interactive':
        interactive_mode()
    else:
        demo_scenario()

if __name__ == "__main__":
    main()
