# Interview Chatbot

An LLM-powered adaptive interview system that generates questions across multiple archetypes (knowledge, reasoning, situational, behavioural, case), evaluates answers, and routes follow-ups — all driven by a declarative domain taxonomy YAML.

## Architecture

```
main.go → handler/ (HTTP) → dispatcher/ (generators) → LLM clients
                          → evaluator/ (scoring)
                          → followup/ (routing)
                          → session/ (state)
                          → utils/ (tree walk + selector)
```

**Generators** (registered in `main.go`):

| Archetype | Source | Generator | Description |
|-----------|--------|-----------|-------------|
| knowledge | `kb_prompt` | `KnowledgeGenerator` | LLM generates factual questions from knowledge context |
| reasoning | `parametric` | `ReasoningGenerator` | Template-based puzzles (sequence, time, logic) |
| situational | `slot_fill` | `SituationalGenerator` | LLM generates scenario questions from role/constraint/stakeholder slots |
| behavioural | `star` | `BehaviouralGenerator` | Pre-written STAR-method questions from competencies |
| case | `free_llm` | `CaseGenerator` | LLM generates case study questions |

**LLM Providers**: Gemini, OpenAI, Ollama, NVIDIA (selectable via `LLM_PROVIDER`).

## Quick Start

```bash
# 1. Configure
cp .env.example .env
# Edit .env with your API key

# 2. Run server
go build -o questionnaire-server . && ./questionnaire-server

# 3. Open browser
open http://localhost:9090

# 4. Or run automated eval
python3 scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml
```

## Configuration

All via environment variables (`.env`):

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `ollama` | One of `gemini`, `openai`, `ollama`, `nvidia` |
| `PORT` | `8080` | HTTP server port |
| `LLM_TIMEOUT_SECONDS` | `120` | HTTP client timeout for LLM API calls |
| `*_API_KEY` | — | API key for the chosen provider |
| `*_MODEL` | varies | Model name for the chosen provider |
| `*_MAX_KNOWLEDGE_TOKENS` | `2000` | Token limit for knowledge context truncation |

### Providers

- **NVIDIA**: `meta/llama-3.1-70b-instruct` via `https://integrate.api.nvidia.com/v1`
- **OpenAI**: `gpt-4o` via `https://api.openai.com/v1`
- **Gemini**: `gemini-1.5-flash` via Google AI
- **Ollama**: `llama3` via local `http://localhost:11434`

## Domain YAML

Define interview structure in a YAML file. Example at `domains/example_domain.yaml`:

```yaml
domain_id: "my_domain"
framework_prompt: "You are an expert interviewer in..."
knowledge_context: |
  # Topic
  - Key concept 1
  - Key concept 2
taxonomy:
  id: "root"
  label: "Root"
  weight: 1.0
  children:
    - id: "topic"
      label: "Topic Name"
      weight: 0.6
      children:
        - id: "topic.subtopic"
          label: "Subtopic"
          weight: 1.0
          archetype_mix:
            - archetype: "knowledge"
              source: "kb_prompt"
              count: 2
          generation_prompt: "Focus on..."
          follow_up_flag: true
          max_follow_ups: 2
competencies:
  - "teamwork"
  - "problem solving"
situational_slots:
  role: ["backend developer"]
  constraint: ["legacy system"]
  stakeholder: ["end users"]
  pressure: ["production outage"]
```

### Archetype Mix Sources

| `source` | Generator | Purpose |
|----------|-----------|---------|
| `kb_prompt` | KnowledgeGenerator | LLM generates from knowledge_context |
| `parametric` | ReasoningGenerator | Template-based (templates/reasoning/) |
| `slot_fill` | SituationalGenerator | LLM generates from situational_slots + template slots |
| `star` | BehaviouralGenerator | Pre-written competency questions |
| `free_llm` | CaseGenerator | LLM generates free-form case studies |

### Sibling Weights

Weights at each level must sum to 1.0. Leaf scores are weighted by the product of ancestor weights.

### Follow-up Templates

```yaml
follow_up_templates:
  - id: "my_followup"
    trigger: "score_low"    # one of: score_low, vague, always, caps_remain
    text: "Can you elaborate on {{answer}}?"
```

## API

### `POST /sessions/init`

```json
{"domain_config_path": "domains/example_domain.yaml"}
```

Returns session with question pool and first asked question.

### `POST /sessions/:id/turn`

```json
{"answer": "Your answer text here"}
```

Returns eval result (score 1-5, concepts, reasoning), optional follow-up, next question.

### `GET /sessions/:id/summary?include_narrative=true`

Returns leaf scores, overall score, transcript, and optional LLM-generated narrative.

## UI

Single-page HTML at `ui/index.html`, served at `GET /` and `GET /ui/`.

Features:
- Configurable server URL and domain config path (setup bar)
- Chat bubbles with archetype-colored badges
- Answer input with Enter-to-send
- Evaluation cards with score dots, covered/missing concepts
- Follow-up indicators
- Summary panel (auto-loads on interview completion)
- Stop/Start controls

## Eval Script

```bash
python3 scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml
python3 scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml --answers answers.txt
python3 scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml --interactive
python3 scripts/eval.py --server http://localhost:9090 --domain domains/example_domain.yaml --no-narrative
```

Auto-answers with archetype-specific responses by default. Supports canned answers from file or interactive mode.

## Development

```bash
go build ./... && go vet ./... && go test ./... -count=1
```

- 50+ unit tests across all packages
- `go vet` clean
- Retry logic with exponential backoff + jitter (handles 429 rate limits)
- Configurable LLM timeouts
