package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/example/raspberry-api/internal/domain"
)

var requiredColumns = []string{"year", "title", "studios", "producers", "winner"}

func LoadFile(path string) ([]domain.Movie, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abrindo csv: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

func Parse(r io.Reader) ([]domain.Movie, error) {
	cr := csv.NewReader(r)
	cr.Comma = ';'
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true

	head, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("lendo cabeçalho: %w", err)
	}
	idx, err := mapHeader(head)
	if err != nil {
		return nil, err
	}

	var movies []domain.Movie
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		line, _ := cr.FieldPos(0)
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", line, err)
		}

		cell := func(col string) string {
			i := idx[col]
			if i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}

		if allBlank(rec) {
			continue
		}
		year, err := strconv.Atoi(cell("year"))
		if err != nil {
			return nil, fmt.Errorf("linha %d: ano inválido %q", line, cell("year"))
		}
		winner, err := parseWinner(cell("winner"))
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", line, err)
		}
		movies = append(movies, domain.Movie{
			Year:      year,
			Title:     cell("title"),
			Studios:   cell("studios"),
			Producers: cell("producers"),
			Winner:    winner,
		})
	}
	return movies, nil
}

func mapHeader(head []string) (map[string]int, error) {
	idx := map[string]int{}
	for i, h := range head {
		h = strings.TrimPrefix(h, "\ufeff") // BOM
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, c := range requiredColumns {
		if _, ok := idx[c]; !ok {
			return nil, fmt.Errorf("cabeçalho sem a coluna obrigatória %q", c)
		}
	}
	return idx, nil
}

func parseWinner(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "yes":
		return true, nil
	case "", "no":
		return false, nil
	default:
		return false, fmt.Errorf("valor inválido para winner: %q (esperado 'yes', 'no' ou vazio)", s)
	}
}

func allBlank(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}
