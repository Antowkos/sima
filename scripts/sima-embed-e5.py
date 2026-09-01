#!/usr/bin/env python3
"""SIMA embedding helper for brief retrieval.

Reads JSON from stdin:
  {"model":"intfloat/multilingual-e5-small","texts":[{"id":"...","text":"..."}]}

Writes JSON to stdout:
  {"embeddings":[{"id":"...","vector":[...]}]}

Install optional dependency:
  python3 -m pip install sentence-transformers
"""

from __future__ import annotations

import json
import sys

try:
    from sentence_transformers import SentenceTransformer
except ImportError as exc:  # pragma: no cover - optional helper script
    raise SystemExit(
        "sentence-transformers is required. Install with: "
        "python3 -m pip install sentence-transformers"
    ) from exc


def prefix_for_e5(text_id: str, text: str) -> str:
    text = " ".join((text or "").split())
    if text_id == "__task__":
        return "query: " + text
    return "passage: " + text


def main() -> int:
    request = json.load(sys.stdin)
    model_name = request.get("model") or "intfloat/multilingual-e5-small"
    texts = request.get("texts") or []
    model = SentenceTransformer(model_name)
    inputs = [prefix_for_e5(item.get("id", ""), item.get("text", "")) for item in texts]
    vectors = model.encode(inputs, normalize_embeddings=True).tolist()
    response = {
        "embeddings": [
            {"id": item.get("id", ""), "vector": vector}
            for item, vector in zip(texts, vectors)
        ]
    }
    json.dump(response, sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
