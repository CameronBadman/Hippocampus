import boto3
import json
import requests

bedrock = boto3.client('bedrock-runtime', region_name='us-east-1')

HIPPOCAMPUS_API = "https://rbf04f5hud.execute-api.ap-southeast-2.amazonaws.com"

def test_agent_curate():
    print("🤖 AI Agent → AI Agent Curation Demo")
    print("=" * 70)
    print("\nThis demonstrates an AI agent calling Hippocampus's internal")
    print("AI agent to curate information autonomously.\n")
    
    system_prompt = """You are a personal assistant that helps users manage their information.

When a user shares personal information with you, use the agent_curate tool to 
store it in Hippocampus. The database has an internal AI agent that will 
intelligently decompose the information into searchable memories.

When asked questions about the user, use search_memory to retrieve relevant information.

You decide:
- Which information is important enough to store
- The importance level (high/medium/low)
- How much delay between insertions (timeout_ms)
- Search parameters (epsilon, threshold, top_k)

Be conversational and explain what you're doing."""

    tools = [
        {
            "toolSpec": {
                "name": "agent_curate",
                "description": "Send information to Hippocampus's internal AI agent for intelligent curation. The agent will decompose text into discrete, searchable memories.",
                "inputSchema": {
                    "json": {
                        "type": "object",
                        "properties": {
                            "agent_id": {
                                "type": "string",
                                "description": "Unique identifier for the user"
                            },
                            "text": {
                                "type": "string",
                                "description": "The information to curate and store"
                            },
                            "importance": {
                                "type": "string",
                                "description": "How thoroughly to extract facts: 'high' (extract everything), 'medium' (key facts), 'low' (critical only)",
                                "enum": ["high", "medium", "low"]
                            },
                            "model_id": {
                                "type": "string",
                                "description": "Which model the internal agent should use",
                                "default": "us.amazon.nova-lite-v1:0"
                            },
                            "bedrock_region": {
                                "type": "string",
                                "description": "AWS region where Bedrock should run",
                                "default": "us-east-1"
                            },
                            "timeout_ms": {
                                "type": "integer",
                                "description": "Milliseconds between each memory insertion (prevents rate limiting)",
                                "default": 50
                            }
                        },
                        "required": ["agent_id", "text", "importance"]
                    }
                }
            }
        },
        {
            "toolSpec": {
                "name": "search_memory",
                "description": "Search for previously stored memories using semantic similarity",
                "inputSchema": {
                    "json": {
                        "type": "object",
                        "properties": {
                            "agent_id": {
                                "type": "string",
                                "description": "User identifier"
                            },
                            "text": {
                                "type": "string",
                                "description": "What to search for"
                            },
                            "epsilon": {
                                "type": "number",
                                "description": "Search radius (0.1-0.5)",
                                "default": 0.3
                            },
                            "threshold": {
                                "type": "number",
                                "description": "Minimum similarity (0.0-1.0)",
                                "default": 0.5
                            },
                            "top_k": {
                                "type": "integer",
                                "description": "Maximum results",
                                "default": 5
                            }
                        },
                        "required": ["agent_id", "text"]
                    }
                }
            }
        }
    ]

    def handle_tool_use(tool_name, tool_input):
        if tool_name == "agent_curate":
            print("\n  🔧 Agent calling Hippocampus internal agent...")
            print(f"     Agent ID: {tool_input['agent_id']}")
            print(f"     Importance: {tool_input['importance']}")
            print(f"     Model: {tool_input.get('model_id', 'us.amazon.nova-lite-v1:0')}")
            print(f"     Region: {tool_input.get('bedrock_region', 'us-east-1')}")
            print(f"     Timeout: {tool_input.get('timeout_ms', 50)}ms\n")
            
            response = requests.post(f"{HIPPOCAMPUS_API}/agent-curate", json=tool_input)
            result = response.json()
            
            if response.status_code == 200:
                data = result.get("data", {})
                print(f"  ✓ Internal agent created {data.get('memories_created')} memories:")
                for mem in data.get("memories", [])[:5]:
                    print(f"     • {mem['key']}: {mem['text'][:50]}...")
                if data.get('memories_created', 0) > 5:
                    print(f"     ... and {data.get('memories_created') - 5} more")
                print()
                return result
            else:
                print(f"  ✗ Curation failed: {result.get('error')}\n")
                return {"error": result.get("error")}
        
        elif tool_name == "search_memory":
            print("\n  🔍 Agent searching memories...")
            print(f"     Query: {tool_input['text']}")
            print(f"     Parameters: epsilon={tool_input.get('epsilon', 0.3)}, threshold={tool_input.get('threshold', 0.5)}, top_k={tool_input.get('top_k', 5)}\n")
            
            response = requests.post(f"{HIPPOCAMPUS_API}/search", json=tool_input)
            result = response.json()
            
            if response.status_code == 200:
                memories = result.get("data", [])
                print(f"  ✓ Found {len(memories)} relevant memories:")
                for mem in memories[:3]:
                    print(f"     • {mem}")
                if len(memories) > 3:
                    print(f"     ... and {len(memories) - 3} more")
                print()
                return {"memories": memories, "count": len(memories)}
            else:
                print(f"  ✗ Search failed\n")
                return {"error": result.get("error"), "memories": []}
        
        return {"error": "Unknown tool"}

    def chat(user_message, conversation_history):
        conversation_history.append({
            "role": "user",
            "content": [{"text": user_message}]
        })
        
        response = bedrock.converse(
            modelId="us.amazon.nova-lite-v1:0",
            messages=conversation_history,
            system=[{"text": system_prompt}],
            toolConfig={"tools": tools}
        )
        
        while response['stopReason'] == 'tool_use':
            tool_requests = [c for c in response['output']['message']['content'] if 'toolUse' in c]
            
            tool_results = []
            for tool_request in tool_requests:
                tool_use = tool_request['toolUse']
                result = handle_tool_use(tool_use['name'], tool_use['input'])
                
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
                system=[{"text": system_prompt}],
                toolConfig={"tools": tools}
            )
        
        assistant_message = response['output']['message']
        conversation_history.append(assistant_message)
        
        return assistant_message['content'][0]['text']

    conversation = []
    
    print("=" * 70)
    print("PART 1: Agent-to-Agent Curation")
    print("=" * 70)
    
    user_input_1 = """Hey, I wanted to update you on my life. My name is Sarah Chen, 
I'm 34 years old, and I work as a software engineer at Google focusing on cloud 
infrastructure. I live in Seattle with my husband Michael, who teaches high school 
math. We have two kids - Emma is 5 and has a severe peanut allergy, and Jake is 8 
and plays competitive soccer. I'm currently training for the Seattle marathon in 
November, running about 40 miles a week. I'm mostly vegetarian but eat fish 
occasionally, and I'm allergic to shellfish and latex. I went to MIT for undergrad 
and got my master's at Stanford. My parents live in San Francisco, and I try to 
visit them monthly. I love sci-fi novels - currently reading the Three Body Problem 
series. I speak Mandarin fluently and I'm learning Spanish with the kids. We have 
a golden retriever named Cosmo who's 3 years old. I'm trying to quit coffee, down 
to one cup in the morning. My favorite programming language is Rust, but I mostly 
work in Go and Python at Google. Oh, and I've been taking piano lessons for about 
2 years now."""
    
    print(f"\nUser: {user_input_1[:150]}...\n")
    
    response = chat(user_input_1, conversation)
    
    print(f"Assistant: {response}")
    
    input("\n[Press Enter to continue...]")
    
    print("\n" + "=" * 70)
    print("PART 2: Querying the Curated Memories")
    print("=" * 70)
    
    queries = [
        "What programming languages does Sarah use?",
        "Tell me about Sarah's family",
        "Does Sarah have any allergies I should know about?"
    ]
    
    for i, query in enumerate(queries, 1):
        print(f"\nQuery {i}: {query}\n")
        response = chat(query, conversation)
        print(f"Assistant: {response}")
        
        if i < len(queries):
            input("\n[Press Enter for next query...]")
    
    print("\n" + "=" * 70)
    print("✅ DEMONSTRATION COMPLETE")
    print("=" * 70)
    print("\nWhat just happened:")
    print("  1. External AI agent (Bedrock Nova) received user information")
    print("  2. Agent decided to use agent_curate tool")
    print("  3. Hippocampus's internal AI agent analyzed the text")
    print("  4. Internal agent decomposed info into discrete memories")
    print("  5. External agent then asked specific questions")
    print("  6. Retrieved precise, relevant memories for each query")
    print("\n💡 This is AI agents orchestrating AI agents - autonomous curation!")
    print("=" * 70)

if __name__ == "__main__":
    test_agent_curate()
