from sentence_transformers import SentenceTransformer

_model = None

def _get_model():
    global _model
    if _model is None:
        _model = SentenceTransformer("intfloat/multilingual-e5-base")
    return _model

def embed(texts: list[str]) -> list[list[float]]:
    """Без префиксов — Go уже добавил passage:/query:."""
    model = _get_model()
    vecs = model.encode(texts, normalize_embeddings=True, batch_size=16)
    return vecs.tolist()