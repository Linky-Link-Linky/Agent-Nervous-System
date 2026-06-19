# ANS Python SDK

Cryptographic audit receipts for every AI agent tool call

## Install

```bash
pip install ans-sdk           # core (zero dependencies)
pip install "ans-sdk[tui]"    # with TUI companion
pip install "ans-sdk[all]"    # with all integrations
```

## Quick Start

```python
import ans
ans.configure("your-agent-id")

@ans.trace(action_type="file.write")
def write_file(path, content):
    with open(path, 'w') as f:
        f.write(content)

write_file("test.txt", "hello")
```

Then run `ans chain` to see the signed receipt tree, or `ans-tui` for the interactive dashboard.

## Direct Client API

```python
from ans import ANSClient

client = ANSClient()
client.register(name="my-agent")
receipt = client.sign_append(
    agent_id="ans_abc123",
    phase="pre",
    action_type="custom",
    payload={"key": "value"},
)
result = client.verify(receipt["receipt_id"])
print(result)  # {"valid": true, ...}
status = client.status()
client.ping()
```

## Integrations

```bash
pip install "ans-sdk[anthropic]"       # Anthropic Messages API
pip install "ans-sdk[openai]"           # OpenAI Agents SDK
pip install "ans-sdk[langchain]"        # LangChain callbacks
pip install "ans-sdk[langgraph]"        # LangGraph callbacks
pip install "ans-sdk[crewai]"           # CrewAI tools
pip install "ans-sdk[google-genai]"     # Google Generative AI
pip install "ans-sdk[mcp]"             # MCP middleware
pip install "ans-sdk[pydantic-ai]"     # Pydantic AI
pip install "ans-sdk[ollama]"          # Ollama
```

See `ans/integrations/` for usage examples.

## TUI

Launch the interactive terminal dashboard:

```bash
ans-tui
# or: python -m ans.tui
```

Features: receipt chain tree, agent table, receipt verification, daemon status.

## Daemon

The ANS daemon (`ans start/stop/status`) is a single Go binary — see the main project README for installation.
