#!/bin/sh
set -eu
export OLLAMA_HOST=0.0.0.0:${PORT:-11434}
MODEL="${OLLAMA_MODEL:-llama3.2:1b}"

ollama serve &
pid=$!
sleep 3
ollama pull "$MODEL" || true
wait "$pid"
