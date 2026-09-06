#!/bin/sh
set -eu
export OLLAMA_HOST=0.0.0.0:${PORT:-11434}
MODEL="${OLLAMA_MODEL:-qwen2.5:0.5b}"

ollama serve &
pid=$!
sleep 3
ollama pull "$MODEL" || true
wait "$pid"
