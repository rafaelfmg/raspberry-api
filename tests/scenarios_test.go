// Testes de integração com cenários variados de dados de entrada. Cada teste
// grava um CSV temporário, sobe a aplicação completa (banco em memória + carga
// + HTTP) e valida a resposta real da API.
package tests

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/example/raspberry-api/internal/app"
	"github.com/example/raspberry-api/internal/domain"
)

const csvHeader = "year;title;studios;producers;winner\n"

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func writeCSV(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "movies.csv")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func serverFromCSV(t *testing.T, content string) *httptest.Server {
	t.Helper()
	a, err := app.New(context.Background(), writeCSV(t, content), discardLogger())
	if err != nil {
		t.Fatalf("iniciando aplicação: %v", err)
	}
	srv := httptest.NewServer(a.Handler)
	t.Cleanup(func() { srv.Close(); a.Close() })
	return srv
}

func fetchIntervals(t *testing.T, content string) domain.IntervalsResponse {
	t.Helper()
	srv := serverFromCSV(t, content)
	res := get(t, srv.URL+"/producers/intervals")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, esperado 200", res.StatusCode)
	}
	return decode[domain.IntervalsResponse](t, res)
}

func row(year int, producers, winner string) string {
	return fmt.Sprintf("%d;Film;Studio;%s;%s", year, producers, winner)
}

func rows(lines ...string) string { return csvHeader + strings.Join(lines, "\n") + "\n" }

func iv(producer string, interval, prev, next int) domain.Interval {
	return domain.Interval{Producer: producer, Interval: interval, PreviousWin: prev, FollowingWin: next}
}

func none() []domain.Interval { return []domain.Interval{} }

func TestIntervals_Scenarios(t *testing.T) {
	tests := []struct {
		name    string
		csv     string
		wantMin []domain.Interval
		wantMax []domain.Interval
	}{
		{
			name:    "nenhum filme vencedor",
			csv:     rows(row(2000, "A", ""), row(2001, "B", "no")),
			wantMin: none(), wantMax: none(),
		},
		{
			name:    "ninguem com dois premios",
			csv:     rows(row(2000, "A", "yes"), row(2001, "B", "yes")),
			wantMin: none(), wantMax: none(),
		},
		{
			name:    "um unico produtor com dois premios",
			csv:     rows(row(2000, "A", "yes"), row(2005, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 5, 2000, 2005)},
			wantMax: []domain.Interval{iv("A", 5, 2000, 2005)},
		},
		{
			name:    "min e max de produtores diferentes",
			csv:     rows(row(2000, "A", "yes"), row(2001, "A", "yes"), row(1990, "B", "yes"), row(2010, "B", "yes")),
			wantMin: []domain.Interval{iv("A", 1, 2000, 2001)},
			wantMax: []domain.Interval{iv("B", 20, 1990, 2010)},
		},
		{
			name: "empate em min e em max retorna todos",
			csv: rows(
				row(2000, "A", "yes"), row(2001, "A", "yes"),
				row(2010, "B", "yes"), row(2011, "B", "yes"),
				row(1980, "C", "yes"), row(2000, "C", "yes"),
				row(1960, "D", "yes"), row(1980, "D", "yes"),
			),
			wantMin: []domain.Interval{iv("A", 1, 2000, 2001), iv("B", 1, 2010, 2011)},
			wantMax: []domain.Interval{iv("D", 20, 1960, 1980), iv("C", 20, 1980, 2000)},
		},
		{
			name:    "tres premios: intervalos consecutivos, nao entre extremos",
			csv:     rows(row(1990, "A", "yes"), row(1991, "A", "yes"), row(2010, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 1, 1990, 1991)},
			wantMax: []domain.Interval{iv("A", 19, 1991, 2010)},
		},
		{
			name:    "indicacoes sem vitoria nao entram no calculo",
			csv:     rows(row(2000, "A", "yes"), row(2002, "A", ""), row(2010, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 10, 2000, 2010)},
			wantMax: []domain.Interval{iv("A", 10, 2000, 2010)},
		},
		{
			name:    "linhas fora de ordem cronologica",
			csv:     rows(row(2010, "A", "yes"), row(2000, "A", "yes"), row(2004, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 4, 2000, 2004)},
			wantMax: []domain.Interval{iv("A", 6, 2004, 2010)},
		},
		{
			name: "multiplos produtores por filme (virgula e and)",
			csv: rows(
				row(2000, "A, B and C", "yes"),
				row(2004, "B", "yes"),
				row(2007, "C and D", "yes"),
			),
			wantMin: []domain.Interval{iv("B", 4, 2000, 2004)},
			wantMax: []domain.Interval{iv("C", 7, 2000, 2007)},
		},
		{
			name:    "lista com virgula antes do and",
			csv:     rows(row(2000, "A, B, and C", "yes"), row(2003, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 3, 2000, 2003)},
			wantMax: []domain.Interval{iv("A", 3, 2000, 2003)},
		},
		{
			name:    "and em maiusculas e espacos repetidos",
			csv:     rows(row(2000, "A  Silva AND  B", "yes"), row(2006, "A Silva", "yes")),
			wantMin: []domain.Interval{iv("A Silva", 6, 2000, 2006)},
			wantMax: []domain.Interval{iv("A Silva", 6, 2000, 2006)},
		},
		{
			name:    "nome 'and' dentro de outro nome nao separa",
			csv:     rows(row(2000, "Alexander Salkind", "yes"), row(2003, "Alexander Salkind", "yes")),
			wantMin: []domain.Interval{iv("Alexander Salkind", 3, 2000, 2003)},
			wantMax: []domain.Interval{iv("Alexander Salkind", 3, 2000, 2003)},
		},
		{
			name:    "produtor repetido no mesmo filme conta uma vez",
			csv:     rows(row(2000, "A, A", "yes"), row(2005, "A and A", "yes")),
			wantMin: []domain.Interval{iv("A", 5, 2000, 2005)},
			wantMax: []domain.Interval{iv("A", 5, 2000, 2005)},
		},
		{
			name:    "produtor repetido em filme unico nao gera intervalo",
			csv:     rows(row(2000, "A, A", "yes")),
			wantMin: none(), wantMax: none(),
		},
		{
			name:    "dois premios no mesmo ano geram intervalo zero",
			csv:     rows(row(2000, "A", "yes"), row(2000, "A", "yes"), row(2010, "B", "yes"), row(2012, "B", "yes")),
			wantMin: []domain.Interval{iv("A", 0, 2000, 2000)},
			wantMax: []domain.Interval{iv("B", 2, 2010, 2012)},
		},
		{
			name:    "tres premios no mesmo ano: intervalo identico aparece uma vez",
			csv:     rows(row(2000, "A", "yes"), row(2000, "A", "yes"), row(2000, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 0, 2000, 2000)},
			wantMax: []domain.Interval{iv("A", 0, 2000, 2000)},
		},
		{
			name:    "mesmo produtor em min e max",
			csv:     rows(row(2000, "A", "yes"), row(2001, "A", "yes"), row(2030, "A", "yes")),
			wantMin: []domain.Interval{iv("A", 1, 2000, 2001)},
			wantMax: []domain.Interval{iv("A", 29, 2001, 2030)},
		},
		{
			name:    "so cabecalho, sem filmes",
			csv:     csvHeader,
			wantMin: none(), wantMax: none(),
		},
		{
			name:    "winner em qualquer caixa",
			csv:     rows(row(2000, "A", "YES"), row(2002, "A", "Yes"), row(2003, "A", "yEs")),
			wantMin: []domain.Interval{iv("A", 1, 2002, 2003)},
			wantMax: []domain.Interval{iv("A", 2, 2000, 2002)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fetchIntervals(t, tc.csv)
			if !reflect.DeepEqual(got.Min, tc.wantMin) {
				t.Errorf("min\nobtido:   %+v\nesperado: %+v", got.Min, tc.wantMin)
			}
			if !reflect.DeepEqual(got.Max, tc.wantMax) {
				t.Errorf("max\nobtido:   %+v\nesperado: %+v", got.Max, tc.wantMax)
			}
		})
	}
}

func TestLoader_FormatVariations(t *testing.T) {
	csv := "\ufeffWinner;Producers;YEAR;Studios;Title\r\n" +
		"YES; A ;2000;S;T1\r\n" +
		"\r\n" +
		"Yes;A;2005;S;T2\r\n" +
		";B;2006;S;T3\r\n" +
		"no;B;2007;S;T4\r\n"
	srv := serverFromCSV(t, csv)

	if all := decode[[]domain.Movie](t, get(t, srv.URL+"/movies")); len(all) != 4 {
		t.Errorf("filmes: %d, esperado 4", len(all))
	}
	if w := decode[[]domain.Movie](t, get(t, srv.URL+"/movies?winner=true")); len(w) != 2 {
		t.Errorf("vencedores: %d, esperado 2", len(w))
	}
	got := decode[domain.IntervalsResponse](t, get(t, srv.URL+"/producers/intervals"))
	want := []domain.Interval{iv("A", 5, 2000, 2005)}
	if !reflect.DeepEqual(got.Min, want) || !reflect.DeepEqual(got.Max, want) {
		t.Errorf("resposta inesperada: %+v", got)
	}

	// Linha sem a coluna winner (menos campos que o cabeçalho) = não vencedor.
	short := rows(row(2000, "A", "yes"), "2004;Film;Studio;A")
	srv2 := serverFromCSV(t, short)
	if w := decode[[]domain.Movie](t, get(t, srv2.URL+"/movies?winner=true")); len(w) != 1 {
		t.Errorf("vencedores: %d, esperado 1", len(w))
	}
}

func TestApp_FailsOnInvalidData(t *testing.T) {
	tests := []struct {
		name, csv, wantErr string
	}{
		{"ano invalido", rows(row(2000, "A", "yes"), "abc;Film;S;A;yes"), "ano inválido"},
		{"ano vazio", rows(";Film;S;A;yes"), "ano inválido"},
		{"winner invalido", rows("2000;Film;S;A;talvez"), "winner"},
		{"coluna ausente", "year;title;studios;producers\n2000;F;S;A\n", "winner"},
		{"arquivo vazio", "", "cabeçalho"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, err := app.New(context.Background(), writeCSV(t, tc.csv), discardLogger())
			if err == nil {
				a.Close()
				t.Fatal("esperado erro na inicialização")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("erro %q não menciona %q", err, tc.wantErr)
			}
		})
	}

	if _, err := app.New(context.Background(), filepath.Join(t.TempDir(), "nao-existe.csv"), discardLogger()); err == nil {
		t.Error("esperado erro para arquivo inexistente")
	}
}

type genMovie struct {
	year      int
	producers []string
	winner    bool
}

func oracle(movies []genMovie) domain.IntervalsResponse {
	years := map[string][]int{}
	for _, m := range movies {
		if !m.winner {
			continue
		}
		seen := map[string]bool{}
		for _, p := range m.producers {
			if !seen[p] {
				seen[p] = true
				years[p] = append(years[p], m.year)
			}
		}
	}
	set := map[domain.Interval]bool{}
	for p, ys := range years {
		sort.Ints(ys)
		for i := 1; i < len(ys); i++ {
			set[iv(p, ys[i]-ys[i-1], ys[i-1], ys[i])] = true
		}
	}
	res := domain.IntervalsResponse{Min: none(), Max: none()}
	if len(set) == 0 {
		return res
	}
	lo, hi := 1<<30, -1
	for v := range set {
		if v.Interval < lo {
			lo = v.Interval
		}
		if v.Interval > hi {
			hi = v.Interval
		}
	}
	for v := range set {
		if v.Interval == lo {
			res.Min = append(res.Min, v)
		}
		if v.Interval == hi {
			res.Max = append(res.Max, v)
		}
	}
	less := func(s []domain.Interval) {
		sort.Slice(s, func(i, j int) bool {
			a, b := s[i], s[j]
			if a.PreviousWin != b.PreviousWin {
				return a.PreviousWin < b.PreviousWin
			}
			if a.Producer != b.Producer {
				return a.Producer < b.Producer
			}
			return a.FollowingWin < b.FollowingWin
		})
	}
	less(res.Min)
	less(res.Max)
	return res
}

func generate(seed int64) ([]genMovie, string) {
	r := rand.New(rand.NewSource(seed))
	pool := 2 + r.Intn(9)
	n := 1 + r.Intn(80)
	base, span := 1980, 1+r.Intn(45)
	yesWords := []string{"yes", "YES", "Yes"}

	var movies []genMovie
	var lines []string
	for i := 0; i < n; i++ {
		k := 1 + r.Intn(3)
		names := make([]string, k)
		for j := range names {
			names[j] = fmt.Sprintf("Producer %d", r.Intn(pool)) // pode repetir no mesmo filme
		}
		var prod string
		switch {
		case k == 1:
			prod = names[0]
		case k == 2 && r.Intn(2) == 0:
			prod = names[0] + " and " + names[1]
		case k == 2:
			prod = names[0] + ", " + names[1]
		case r.Intn(2) == 0:
			prod = names[0] + ", " + names[1] + " and " + names[2]
		default:
			prod = names[0] + ", " + names[1] + ", and " + names[2]
		}
		m := genMovie{year: base + r.Intn(span), producers: names, winner: r.Float64() < 0.35}
		winner := ""
		if m.winner {
			winner = yesWords[r.Intn(len(yesWords))]
		} else if r.Intn(4) == 0 {
			winner = "no"
		}
		movies = append(movies, m)
		lines = append(lines, fmt.Sprintf("%d;Movie %d;Studio;%s;%s", m.year, i, prod, winner))
	}
	return movies, rows(lines...)
}

func TestIntervals_RandomDatasetsMatchOracle(t *testing.T) {
	for seed := int64(1); seed <= 150; seed++ {
		movies, csv := generate(seed)
		want := oracle(movies)
		got := fetchIntervals(t, csv)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d\nobtido:   %+v\nesperado: %+v\ncsv:\n%s", seed, got, want, csv)
		}
	}
}
