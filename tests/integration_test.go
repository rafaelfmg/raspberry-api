// Testes de integração: sobem a aplicação completa (SQLite em memória + carga
// do CSV oficial + HTTP) e validam as respostas contra os dados fornecidos.
package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/example/raspberry-api/internal/app"
	"github.com/example/raspberry-api/internal/domain"
)

const csvPath = "../data/movielist.csv"

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	a, err := app.New(context.Background(), csvPath, log)
	if err != nil {
		t.Fatalf("iniciando aplicação: %v", err)
	}
	srv := httptest.NewServer(a.Handler)
	t.Cleanup(func() { srv.Close(); a.Close() })
	return srv
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatalf("decodificando JSON: %v", err)
	}
	return v
}

func TestProducersIntervals_MatchesProvidedData(t *testing.T) {
	srv := newServer(t)
	res := get(t, srv.URL+"/producers/intervals")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, esperado 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type %q, esperado application/json", ct)
	}

	got := decode[domain.IntervalsResponse](t, res)
	want := domain.IntervalsResponse{
		Min: []domain.Interval{{Producer: "Joel Silver", Interval: 1, PreviousWin: 1990, FollowingWin: 1991}},
		Max: []domain.Interval{{Producer: "Matthew Vaughn", Interval: 13, PreviousWin: 2002, FollowingWin: 2015}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("resposta diferente da esperada\nobtido:   %+v\nesperado: %+v", got, want)
	}
}

func TestProducersIntervals_JSONShape(t *testing.T) {
	srv := newServer(t)
	var raw map[string][]map[string]any
	if err := json.NewDecoder(get(t, srv.URL+"/producers/intervals").Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	for _, list := range []string{"min", "max"} {
		items, ok := raw[list]
		if !ok || len(items) == 0 {
			t.Fatalf("chave %q ausente ou vazia", list)
		}
		for _, key := range []string{"producer", "interval", "previousWin", "followingWin"} {
			if _, ok := items[0][key]; !ok {
				t.Errorf("%s[0] sem a chave %q", list, key)
			}
		}
	}
}

func TestMovies_LoadedFromCSV(t *testing.T) {
	srv := newServer(t)

	all := decode[[]domain.Movie](t, get(t, srv.URL+"/movies"))
	if len(all) != 206 {
		t.Errorf("filmes carregados: %d, esperado 206", len(all))
	}
	winners := decode[[]domain.Movie](t, get(t, srv.URL+"/movies?winner=true"))
	if len(winners) != 42 {
		t.Errorf("vencedores: %d, esperado 42", len(winners))
	}
	for _, m := range winners {
		if !m.Winner {
			t.Errorf("filme %q retornado em winner=true mas Winner=false", m.Title)
		}
	}

	// Primeiro registro do CSV fornecido.
	first := all[0]
	if first.Year != 1980 || first.Title != "Can't Stop the Music" ||
		first.Producers != "Allan Carr" || !first.Winner {
		t.Errorf("primeiro filme inesperado: %+v", first)
	}

	y1980 := decode[[]domain.Movie](t, get(t, srv.URL+"/movies?year=1980&winner=true"))
	if len(y1980) != 1 || y1980[0].Title != "Can't Stop the Music" {
		t.Errorf("vencedor de 1980 inesperado: %+v", y1980)
	}
}

func TestMovies_GetByID(t *testing.T) {
	srv := newServer(t)
	res := get(t, srv.URL+"/movies/1")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, esperado 200", res.StatusCode)
	}
	if m := decode[domain.Movie](t, res); m.ID != 1 || m.Year != 1980 {
		t.Errorf("filme inesperado: %+v", m)
	}
}

func TestHTTPStatusCodes(t *testing.T) {
	srv := newServer(t)
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/movies/99999", http.StatusNotFound},
		{http.MethodGet, "/movies/abc", http.StatusBadRequest},
		{http.MethodGet, "/movies?year=abc", http.StatusBadRequest},
		{http.MethodGet, "/movies?winner=talvez", http.StatusBadRequest},
		{http.MethodPost, "/producers/intervals", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/movies", http.StatusMethodNotAllowed},
		{http.MethodGet, "/inexistente", http.StatusNotFound},
	} {
		req, _ := http.NewRequest(tc.method, srv.URL+tc.path, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Errorf("%s %s: status %d, esperado %d", tc.method, tc.path, res.StatusCode, tc.want)
		}
	}
}
