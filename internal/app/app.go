// Package app monta a aplicação (banco em memória + carga do CSV + HTTP),
// permitindo que o mesmo wiring seja usado pelo main e pelos testes de integração.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/example/raspberry-api/internal/httpapi"
	"github.com/example/raspberry-api/internal/loader"
	"github.com/example/raspberry-api/internal/repository"
)

type App struct {
	Handler http.Handler
	repo    *repository.Repository
}

func New(ctx context.Context, csvPath string, log *slog.Logger) (*App, error) {
	repo, err := repository.NewInMemory(ctx)
	if err != nil {
		return nil, fmt.Errorf("criando banco: %w", err)
	}
	movies, err := loader.LoadFile(csvPath)
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("lendo csv: %w", err)
	}
	if err := repo.BulkInsert(ctx, movies); err != nil {
		repo.Close()
		return nil, fmt.Errorf("carregando dados: %w", err)
	}
	log.Info("carga concluída", "arquivo", csvPath, "filmes", len(movies))
	return &App{Handler: httpapi.New(repo, log), repo: repo}, nil
}

func (a *App) Close() error { return a.repo.Close() }
