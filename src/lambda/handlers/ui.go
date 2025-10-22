package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

const apiEndpoint = "https://rbf04f5hud.execute-api.ap-southeast-2.amazonaws.com"


var tpl = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Hippocampus Demo UI</title>
<style>
	body { font-family: Arial, sans-serif; margin: 20px; background: #f9f9f9; color: #333; }
	h1, h2 { color: #2c3e50; }
	form { margin-bottom: 30px; padding: 15px; background: #fff; border-radius: 8px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); transition: transform 0.2s ease; }
	input[type=text], textarea { width: 100%; padding: 8px; margin: 5px 0 10px 0; border: 1px solid #ccc; border-radius: 4px; }
	button { background-color: #3498db; color: white; padding: 10px 15px; border: none; border-radius: 4px; cursor: pointer; transition: transform 0.2s ease; }
	button:hover { background-color: #2980b9; transform: scale(1.05); }
	pre { background: #ecf0f1; padding: 10px; border-radius: 5px; overflow-x: auto; min-height: 50px; transition: transform 0.3s ease, opacity 0.3s ease; }

	/* Bounce animation */
	@keyframes bounce {
		0% { transform: translateY(0); }
		25% { transform: translateY(-10px); }
		50% { transform: translateY(0); }
		75% { transform: translateY(-5px); }
		100% { transform: translateY(0); }
	}

	.result-bounce {
		animation: bounce 0.5s;
	}
</style>
</head>
<body>
<h1>Hippocampus Demo UI</h1>

<h2>Insert Memory</h2>
<form id="insertForm">
Agent ID: <input name="agent_id" value="safety_demo_parent"><br>
Key: <input name="key"><br>
Text: <textarea name="text"></textarea><br>
<button type="submit">Insert</button>
</form>

<h2>Query Safety Agent</h2>
<form id="safetyForm">
Agent ID: <input name="agent_id" value="safety_demo_parent"><br>
Message: <textarea name="message"></textarea><br>
<button type="submit">Query</button>
</form>

<h3>Result</h3>
<pre id="result"></pre>

<script>
async function postJSON(path, data) {
	const res = await fetch(path, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	return await res.text();
}

function showResult(text) {
	const resultEl = document.getElementById('result');
	resultEl.textContent = text;
	resultEl.classList.remove('result-bounce');
	void resultEl.offsetWidth; // trigger reflow
	resultEl.classList.add('result-bounce');
}

document.getElementById('insertForm').addEventListener('submit', async e => {
	e.preventDefault();
	const data = {
		agent_id: e.target.agent_id.value,
		key: e.target.key.value,
		text: e.target.text.value
	};
	showResult("Sending...");
	const result = await postJSON('/insert', data);
	showResult(result);
});

document.getElementById('safetyForm').addEventListener('submit', async e => {
	e.preventDefault();
	const data = {
		agent_id: e.target.agent_id.value,
		message: e.target.message.value
	};
	showResult("Sending...");
	const result = await postJSON('/agent-safety', data);
	showResult(result);
});
</script>

</body>
</html>
`))



type SafetyRequest struct {
	AgentID string `json:"agent_id"`
	Message string `json:"message"`
}

type UIData struct {
	Result string
}

func (h *Handler) HandleUI(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// GET request → render empty page
	if strings.ToUpper(request.HTTPMethod) == "GET" {
		var buf bytes.Buffer
		tpl.Execute(&buf, nil)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       buf.String(),
			Headers: map[string]string{
				"Content-Type": "text/html",
			},
		}, nil
	}

	// POST → parse form values (application/x-www-form-urlencoded)
	values := map[string]string{}
	for _, pair := range strings.Split(request.Body, "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key, _ := url.QueryUnescape(kv[0])
			val, _ := url.QueryUnescape(kv[1])
			values[key] = val
		}
	}

	var endpoint string
	var payload []byte
	switch request.Path {
	case "/insert":
		req := InsertRequest{
			AgentID: values["agent_id"],
			Key:     values["key"],
			Text:    values["text"],
		}
		payload, _ = json.Marshal(req)
		endpoint = apiEndpoint + "/insert"
	case "/agent-safety":
		req := SafetyRequest{
			AgentID: values["agent_id"],
			Message: values["message"],
		}
		payload, _ = json.Marshal(req)
		endpoint = apiEndpoint + "/agent-safety"
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       "Unknown endpoint",
		}, nil
	}

	// Call backend
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to call backend: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var buf bytes.Buffer
	tpl.Execute(&buf, UIData{Result: string(body)})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       buf.String(),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil
}
