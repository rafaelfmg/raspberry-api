package service

import (
	"regexp"
	"sort"
	"strings"

	"github.com/example/raspberry-api/internal/domain"
)

var andSeparator = regexp.MustCompile(`(?i)\s+and\s+`)

func SplitProducers(s string) []string {
	s = andSeparator.ReplaceAllString(s, ",")
	seen := map[string]bool{}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.Join(strings.Fields(p), " ")
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func Intervals(movies []domain.Movie) domain.IntervalsResponse {
	wins := map[string][]int{}
	for _, m := range movies {
		if !m.Winner {
			continue
		}
		for _, p := range SplitProducers(m.Producers) {
			wins[p] = append(wins[p], m.Year)
		}
	}

	unique := map[domain.Interval]struct{}{}
	for producer, years := range wins {
		if len(years) < 2 {
			continue
		}
		sort.Ints(years)
		for i := 1; i < len(years); i++ {
			unique[domain.Interval{
				Producer:     producer,
				Interval:     years[i] - years[i-1],
				PreviousWin:  years[i-1],
				FollowingWin: years[i],
			}] = struct{}{}
		}
	}

	res := domain.IntervalsResponse{Min: []domain.Interval{}, Max: []domain.Interval{}}
	if len(unique) == 0 {
		return res
	}

	first := true
	var lo, hi int
	for iv := range unique {
		if first || iv.Interval < lo {
			lo = iv.Interval
		}
		if first || iv.Interval > hi {
			hi = iv.Interval
		}
		first = false
	}
	for iv := range unique {
		if iv.Interval == lo {
			res.Min = append(res.Min, iv)
		}
		if iv.Interval == hi {
			res.Max = append(res.Max, iv)
		}
	}

	order := func(s []domain.Interval) {
		sort.Slice(s, func(i, j int) bool {
			if s[i].PreviousWin != s[j].PreviousWin {
				return s[i].PreviousWin < s[j].PreviousWin
			}
			if s[i].Producer != s[j].Producer {
				return s[i].Producer < s[j].Producer
			}
			return s[i].FollowingWin < s[j].FollowingWin
		})
	}
	order(res.Min)
	order(res.Max)
	return res
}
