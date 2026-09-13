#!/usr/bin/env python3
"""OpenAI-compatible local server for Amazon Baby Qwen3 fine-tuned weights."""

from __future__ import annotations

import os
import time
import uuid
from pathlib import Path
from typing import Any

import torch
import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from transformers import AutoModelForCausalLM, AutoTokenizer

DEFAULT_WEIGHTS = (
    Path(__file__).resolve().parent.parent.parent
    / "models"
    / "amazon-baby-qwen3-finetuned"
    / "weights"
)
MODEL_NAME = os.environ.get("LLM_AMAZON_BABY_MODEL_NAME", "miyuna-amazon-baby-qwen3")
PORT = int(os.environ.get("AMAZON_BABY_INFERENCE_PORT", "8765"))
HOST = os.environ.get("AMAZON_BABY_INFERENCE_HOST", "127.0.0.1")

app = FastAPI(title="Amazon Baby Qwen3 Local Inference")

_model: AutoModelForCausalLM | None = None
_tokenizer: AutoTokenizer | None = None
_device: str = "cpu"


class Message(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    model: str = MODEL_NAME
    messages: list[Message]
    max_tokens: int = Field(default=8)
    temperature: float = 0.1


def load_model() -> None:
    global _model, _tokenizer, _device
    weights = Path(os.environ.get("AMAZON_BABY_WEIGHTS_PATH", str(DEFAULT_WEIGHTS)))
    if not (weights / "model.safetensors").exists():
        raise FileNotFoundError(f"Weights not found under {weights}")

    _device = "cuda" if torch.cuda.is_available() else "cpu"
    dtype = torch.bfloat16 if _device == "cuda" else torch.float32

    _tokenizer = AutoTokenizer.from_pretrained(weights, trust_remote_code=True)
    _model = AutoModelForCausalLM.from_pretrained(
        weights,
        torch_dtype=dtype,
        device_map="auto" if _device == "cuda" else None,
        trust_remote_code=True,
    )
    if _device == "cpu":
        _model = _model.to(_device)
    _model.eval()


def build_prompt(messages: list[Message]) -> str:
    parts: list[str] = []
    for msg in messages:
        content = msg.content.strip()
        if not content:
            continue
        if msg.role == "system":
            parts.append(content)
        elif msg.role == "user":
            parts.append(content)
    return "\n\n".join(parts)


@app.on_event("startup")
def startup() -> None:
    load_model()


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "model": MODEL_NAME, "device": _device}


@app.post("/v1/chat/completions")
def chat_completions(req: ChatRequest) -> dict[str, Any]:
    if _model is None or _tokenizer is None:
        raise HTTPException(status_code=503, detail="model not loaded")

    prompt = build_prompt(req.messages)
    inputs = _tokenizer(prompt, return_tensors="pt")
    if _device == "cuda":
        inputs = {k: v.to(_model.device) for k, v in inputs.items()}

    prompt_len = int(inputs["input_ids"].shape[1])
    max_new_tokens = max(1, min(req.max_tokens, 16))

    gen_kwargs: dict[str, Any] = {
        "max_new_tokens": max_new_tokens,
        "pad_token_id": _tokenizer.eos_token_id,
    }
    if req.temperature > 0:
        gen_kwargs["do_sample"] = True
        gen_kwargs["temperature"] = max(req.temperature, 0.01)
    else:
        gen_kwargs["do_sample"] = False

    with torch.no_grad():
        output = _model.generate(**inputs, **gen_kwargs)

    new_tokens = output[0][prompt_len:]
    content = _tokenizer.decode(new_tokens, skip_special_tokens=True).strip()
    completion_tokens = int(new_tokens.shape[0])

    return {
        "id": f"chatcmpl-{uuid.uuid4().hex[:12]}",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": req.model or MODEL_NAME,
        "choices": [
            {
                "index": 0,
                "message": {"role": "assistant", "content": content},
                "finish_reason": "stop",
            }
        ],
        "usage": {
            "prompt_tokens": prompt_len,
            "completion_tokens": completion_tokens,
            "total_tokens": prompt_len + completion_tokens,
        },
    }


if __name__ == "__main__":
    uvicorn.run(app, host=HOST, port=PORT, log_level="info")
