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

	if _, err := cr.Read(); err != nil { // cabeçalho
		return nil, fmt.Errorf("lendo cabeçalho: %w", err)
	}

	var movies []domain.Movie
	for line := 2; ; line++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", line, err)
		}
		if len(rec) < 4 {
			return nil, fmt.Errorf("linha %d: esperado ao menos 4 colunas, obtido %d", line, len(rec))
		}
		year, err := strconv.Atoi(strings.TrimSpace(rec[0]))
		if err != nil {
			return nil, fmt.Errorf("linha %d: ano inválido %q", line, rec[0])
		}
		winner := false
		if len(rec) > 4 {
			winner = strings.EqualFold(strings.TrimSpace(rec[4]), "yes")
		}
		movies = append(movies, domain.Movie{
			Year:      year,
			Title:     strings.TrimSpace(rec[1]),
			Studios:   strings.TrimSpace(rec[2]),
			Producers: strings.TrimSpace(rec[3]),
			Winner:    winner,
		})
	}
	return movies, nil
}
