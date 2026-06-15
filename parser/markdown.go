package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"workout-tracker-go/db"
)

func ParseLine(line string) (string, []db.Serie, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil, fmt.Errorf("Empty line")
	}
	reFinder := regexp.MustCompile(`\d+\s*x\s*\d+`)
	loc := reFinder.FindStringIndex(line)
	if loc == nil {
		return "", nil, fmt.Errorf("No valid series pattern found in line: %s", line)
	}
	exerciseName := strings.TrimSpace(line[:loc[0]])
	rawSeries := strings.TrimSpace(line[loc[0]:])
	var series []db.Serie
	reCompact := regexp.MustCompile(`(\d+)x(\d+)x([\d.]+)`)
	matchesCompact := reCompact.FindAllStringSubmatch(rawSeries, -1)
	if len(matchesCompact) > 0 {
		for _, match := range matchesCompact {
			seriesNumber, _ := strconv.Atoi(match[1])
			reps, _ := strconv.Atoi(match[2])
			weight, _ := strconv.ParseFloat(match[3], 64)
			series = append(series, db.Serie{
				SeriesNumber: seriesNumber,
				Reps:         reps,
				Weight:       weight,
			})
		}
		return exerciseName, series, nil
	}
	reSpaced := regexp.MustCompile(`(\d+)\s*x\s*([\d.]+)`)
	matchesSpaced := reSpaced.FindAllStringSubmatch(rawSeries, -1)
	if len(matchesSpaced) > 0 {
		for idx, match := range matchesSpaced {
			reps, _ := strconv.Atoi(match[1])
			weight, _ := strconv.ParseFloat(match[2], 64)
			series = append(series, db.Serie{
				SeriesNumber: idx + 1,
				Reps:         reps,
				Weight:       weight,
			})
		}
		return exerciseName, series, nil
	}
	return "", nil, fmt.Errorf("Failed to parse series text: %s", rawSeries)
}
