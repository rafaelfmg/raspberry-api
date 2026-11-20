package service

import (
	"sort"
	"strings"

	"github.com/example/raspberry-api/internal/domain"
)

func SplitProducers(s string) []string {
	s = strings.ReplaceAll(s, " and ", ",")
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
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

	var all []domain.Interval
	for producer, years := range wins {
		if len(years) < 2 {
			continue
		}
		sort.Ints(years)
		for i := 1; i < len(years); i++ {
			all = append(all, domain.Interval{
				Producer:     producer,
				Interval:     years[i] - years[i-1],
				PreviousWin:  years[i-1],
				FollowingWin: years[i],
			})
		}
	}

	res := domain.IntervalsResponse{Min: []domain.Interval{}, Max: []domain.Interval{}}
	if len(all) == 0 {
		return res
	}

	lo, hi := all[0].Interval, all[0].Interval
	for _, iv := range all {
		if iv.Interval < lo {
			lo = iv.Interval
		}
		if iv.Interval > hi {
			hi = iv.Interval
		}
	}
	for _, iv := range all {
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
			return s[i].Producer < s[j].Producer
		})
	}
	order(res.Min)
	order(res.Max)
	return res
}
