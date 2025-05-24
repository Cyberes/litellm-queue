# litellm-queue

_A server that sits in front of LiteLLM and queues requests._

[LiteLLM](https://github.com/BerriAI/litellm) has no queueing mechanism for incoming requests, meaning all requests hit
the inference endpoints at the same time. This is fine for most endpoints like OpenAI or Anthropic, but endpoints
running inference servers such as llama.cpp will be quickly overwhelmed.

This is a simple queuing server that sits in front of your LiteLLM server and reads the `model` header of incoming
requests to route them to per-model queues. For example, you can limit the model `gpt-4.1` to 4 concurrent requests or
limit your llama.cpp backend to only 1 concurrent request.

### Install

1. Download the latest release from the [releases tab](https://git.evulid.cc/cyberes/litellm-queue/releases)
2. Copy `config.yaml.service` to `config.yaml`
3. Start the `litellm-queue` server
4. Update your reverse proxy for LiteLLM to point to the listen address of `litellm-queue`

An example systemd service is provided.

### Build

1. `./build.sh`
2. Compiled binary will be at `dist/litellm-queue-0.0.0-...`