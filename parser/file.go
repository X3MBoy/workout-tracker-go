package parser

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type ExtractedCardio struct {
	Duration float64
	Distance float64
	AvgSpeed float64
	MaxSpeed float64
	Calories float64
	MaxHR    int
}

type ExtractedWorkout struct {
	ExerciseName string
	RawSeries    string
}

type DaySession struct {
	Date     string
	Workouts []ExtractedWorkout
	Cardio   *ExtractedCardio
}

func ParseFullMarkdown(filePath string) ([]DaySession, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var sessions []DaySession
	var currentSession *DaySession
	inCardioBlock := false
	scanner := bufio.NewScanner(file)
	dateRegex := regexp.MustCompile(`\b(\d{2}/\d{2}/\d{2,4})\b`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## Día") {
			if currentSession != nil {
				sessions = append(sessions, *currentSession)
			}
			dateStr := "Unknown"
			match := dateRegex.FindStringSubmatch(line)
			if len(match) > 0 {
				dateStr = match[1]
				parts := strings.Split(dateStr, "/")
				if len(parts) == 3 {
					year := parts[2]
					if len(year) == 2 {
						year = "20" + year
					}
					dateStr = year + "-" + parts[1] + "-" + parts[0]
				}
			}
			currentSession = &DaySession{Date: dateStr}
			inCardioBlock = false
			continue
		}
		if currentSession == nil {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			if strings.Contains(strings.ToLower(line), "cardio") {
				inCardioBlock = true
				if currentSession.Cardio == nil {
					currentSession.Cardio = &ExtractedCardio{}
				}
			}
			continue
		}
		if strings.HasPrefix(line, "|") {
			if strings.Contains(strings.ToLower(line), "ejercicio") {
				inCardioBlock = false
				continue
			}
			if strings.Contains(line, "---") || strings.Contains(strings.ToLower(line), "métrica") || strings.Contains(strings.ToLower(line), "valor registrado") {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) < 3 {
				continue
			}
			if inCardioBlock {
				metricName := strings.ToLower(strings.TrimSpace(parts[1]))
				rawVal := strings.TrimSpace(parts[2])
				if strings.Contains(metricName, "tiempo") {
					currentSession.Cardio.Duration = parseRawTimeToMinutes(rawVal)
				} else if strings.Contains(metricName, "distancia") {
					currentSession.Cardio.Distance = extractFloatFromString(rawVal)
				} else if strings.Contains(metricName, "velocidad media") {
					currentSession.Cardio.AvgSpeed = extractFloatFromString(rawVal)
				} else if strings.Contains(metricName, "velocidad máxima") {
					currentSession.Cardio.MaxSpeed = extractFloatFromString(rawVal)
				} else if strings.Contains(metricName, "calorías") || strings.Contains(metricName, "calorias") {
					currentSession.Cardio.Calories = extractFloatFromString(rawVal)
				} else if strings.Contains(metricName, "f. cardíaca") || strings.Contains(metricName, "máximo") {
					currentSession.Cardio.MaxHR = int(extractFloatFromString(rawVal))
				}
				continue
			}
			exName := strings.TrimSpace(parts[1])
			exName = strings.ReplaceAll(exName, "**", "")
			exName = regexp.MustCompile(`^\d+\.\s*`).ReplaceAllString(exName, "")
			exName = strings.TrimSpace(exName)
			var seriesCollected []string
			for i := 2; i < len(parts)-1; i++ {
				cell := strings.TrimSpace(parts[i])
				if cell != "" && !strings.Contains(cell, "kg") && !strings.Contains(cell, "Buscando") && !strings.Contains(cell, "Bloque") {
					seriesCollected = append(seriesCollected, cell)
				}
			}
			rawSeriesCombined := strings.Join(seriesCollected, " ")
			if exName != "" && rawSeriesCombined != "" {
				currentSession.Workouts = append(currentSession.Workouts, ExtractedWorkout{
					ExerciseName: exName,
					RawSeries:    rawSeriesCombined,
				})
			}
		}
	}
	if currentSession != nil {
		sessions = append(sessions, *currentSession)
	}
	return sessions, scanner.Err()
}

func parseRawTimeToMinutes(raw string) float64 {
	reg := regexp.MustCompile(`[^0-9:]`)
	cleanStr := reg.ReplaceAllString(raw, "")
	if cleanStr == "" {
		return 0.0
	}
	if strings.Contains(cleanStr, ":") {
		timeParts := strings.Split(cleanStr, ":")
		if len(timeParts) == 2 {
			m, _ := strconv.ParseFloat(timeParts[0], 64)
			s, _ := strconv.ParseFloat(timeParts[1], 64)
			return m + (s / 60.0)
		} else if len(timeParts) == 3 {
			h, _ := strconv.ParseFloat(timeParts[0], 64)
			m, _ := strconv.ParseFloat(timeParts[1], 64)
			s, _ := strconv.ParseFloat(timeParts[2], 64)
			return (h * 60.0) + m + (s / 60.0)
		}
	}
	val, err := strconv.ParseFloat(cleanStr, 64)
	if err != nil {
		return 0.0
	}
	return val
}

func extractFloatFromString(raw string) float64 {
	re := regexp.MustCompile(`([\d.]+)`)
	match := re.FindString(raw)
	if match != "" {
		val, _ := strconv.ParseFloat(match, 64)
		return val
	}
	return 0
}
