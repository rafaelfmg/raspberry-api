package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/example/raspberry-api/internal/domain"
	"github.com/example/raspberry-api/internal/repository"
	"github.com/example/raspberry-api/internal/service"
)

type MovieStore interface {
	List(ctx context.Context, f domain.Filter) ([]domain.Movie, error)
	GetByID(ctx context.Context, id int64) (domain.Movie, error)
}

type Handler struct {
	store MovieStore
	log   *slog.Logger
}

func New(store MovieStore, log *slog.Logger) http.Handler {
	h := &Handler{store: store, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /movies", h.list)
	mux.HandleFunc("GET /movies/{id}", h.get)
	mux.HandleFunc("GET /producers/intervals", h.intervals)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	var f domain.Filter
	if v := r.URL.Query().Get("year"); v != "" {
		y, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parâmetro 'year' inválido")
			return
		}
		f.Year = &y
	}
	if v := r.URL.Query().Get("winner"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parâmetro 'winner' inválido")
			return
		}
		f.Winner = &b
	}
	movies, err := h.store.List(r.Context(), f)
	if err != nil {
		h.log.Error("listando filmes", "err", err)
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	writeJSON(w, http.StatusOK, movies)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	m, err := h.store.GetByID(r.Context(), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "filme não encontrado")
	case err != nil:
		h.log.Error("buscando filme", "err", err)
		writeError(w, http.StatusInternalServerError, "erro interno")
	default:
		writeJSON(w, http.StatusOK, m)
	}
}

func (h *Handler) intervals(w http.ResponseWriter, r *http.Request) {
	winner := true
	winners, err := h.store.List(r.Context(), domain.Filter{Winner: &winner})
	if err != nil {
		h.log.Error("buscando vencedores", "err", err)
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	writeJSON(w, http.StatusOK, service.Intervals(winners))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
