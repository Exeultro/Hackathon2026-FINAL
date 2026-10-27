package api

import (
	"net/http"

	"aba/internal/mlclient"
)

func SystemHandler(ml *mlclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health, err := ml.Health(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "ml_unavailable", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SystemInfo{
			Offline: true,
			Models: SystemModels{
				ASR:        health["asr"],
				Embeddings: health["embeddings"],
				LLM:        health["llm"],
			},
		})
	}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
