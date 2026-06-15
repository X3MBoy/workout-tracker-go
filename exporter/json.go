package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type IACoachContext struct {
	UserMetadata   *UserMetadataExport `json:"user_metadata"`
	HistorySummary []SessionExport     `json:"history_summary"`
}

type UserMetadataExport struct {
	BirthDate       string  `json:"birth_date"`
	Gender          string  `json:"gender"`
	HeightM         float64 `json:"height_m"`
	InitialWeightKg float64 `json:"initial_weight_kg"`
	CurrentWeightKg float64 `json:"current_weight_kg"`
	LastWeightDate  string  `json:"last_weight_date"`
	HealthContext   string  `json:"health_and_objectives_context"`
}

type SessionExport struct {
	Date     string        `json:"date"`
	Comments string        `json:"comments,omitempty"`
	Cardio   *CardioData   `json:"cardio,omitempty"`
	Workouts []WorkoutData `json:"workouts,omitempty"`
}

type CardioData struct {
	DurationMin float64 `json:"duration_minutes"`
	DistanceKm  float64 `json:"distance_km"`
	AvgSpeed    float64 `json:"avg_speed,omitempty"`
	MaxSpeed    float64 `json:"max_speed,omitempty"`
	Calories    float64 `json:"calories,omitempty"`
	MaxHR       int     `json:"max_heart_rate,omitempty"`
}

type WorkoutData struct {
	Exercise string   `json:"exercise"`
	Series   []string `json:"series"`
}

func ExportToIAJson(dbConn *sql.DB, outputPath string) error {
	contextPayload := IACoachContext{}
	var meta UserMetadataExport
	metaQuery := `
		SELECT birth_date, gender, height_m, initial_weight_kg, current_weight_kg, last_weight_date, health_context 
		FROM user_metadata 
		LIMIT 1`

	err := dbConn.QueryRow(metaQuery).Scan(
		&meta.BirthDate, &meta.Gender, &meta.HeightM,
		&meta.InitialWeightKg, &meta.CurrentWeightKg, &meta.LastWeightDate,
		&meta.HealthContext,
	)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error querying user metadata: %w", err)
	}
	if err == nil {
		contextPayload.UserMetadata = &meta
	}
	sessionQuery := `
		SELECT s.id, s.date_of_exercise, s.comments,
				c.duration_minutes, c.distance_km, c.avg_speed, c.max_speed, c.calories, c.max_heart_rate
		FROM sessions s
		LEFT JOIN cardio_sessions c ON s.id = c.session_id
		ORDER BY s.date_of_exercise ASC`
	rows, err := dbConn.Query(sessionQuery)
	if err != nil {
		return fmt.Errorf("error querying sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sID int64
		var sDate string
		var sComments sql.NullString
		var cDur, cDist, cAvgS, cMaxS, cCal sql.NullFloat64
		var cHR sql.NullInt64
		err := rows.Scan(&sID, &sDate, &sComments, &cDur, &cDist, &cAvgS, &cMaxS, &cCal, &cHR)
		if err != nil {
			return fmt.Errorf("error scanning session row: %w", err)
		}
		session := SessionExport{Date: sDate}
		if sComments.Valid && sComments.String != "" {
			session.Comments = sComments.String
		}
		if cDur.Valid && cDur.Float64 > 0 {
			session.Cardio = &CardioData{
				DurationMin: cDur.Float64,
				DistanceKm:  cDist.Float64,
				AvgSpeed:    cAvgS.Float64,
				MaxSpeed:    cMaxS.Float64,
				Calories:    cCal.Float64,
				MaxHR:       int(cHR.Int64),
			}
		}
		workoutQuery := `
			SELECT e.name, ser.series_number, ser.reps, ser.weight, ser.rpe
			FROM series ser
			JOIN exercises e ON ser.exercise_id = e.id
			WHERE ser.session_id = ?
			ORDER BY ser.exercise_id, ser.series_number ASC`
		wRows, err := dbConn.Query(workoutQuery, sID)
		if err == nil {
			exerciseMap := make(map[string][]string)
			var orderedExercises []string
			for wRows.Next() {
				var exName string
				var serNum, reps int
				var weight float64
				var rpe sql.NullInt64
				if err := wRows.Scan(&exName, &serNum, &reps, &weight, &rpe); err == nil {
					serStr := fmt.Sprintf("%dx%dx%.1f", serNum, reps, weight)
					if rpe.Valid && rpe.Int64 > 0 {
						serStr = fmt.Sprintf("%s (RPE %d)", serStr, rpe.Int64)
					}
					if _, exists := exerciseMap[exName]; !exists {
						orderedExercises = append(orderedExercises, exName)
					}
					exerciseMap[exName] = append(exerciseMap[exName], serStr)
				}
			}
			wRows.Close()
			for _, name := range orderedExercises {
				session.Workouts = append(session.Workouts, WorkoutData{
					Exercise: name,
					Series:   exerciseMap[name],
				})
			}
		}
		contextPayload.HistorySummary = append(contextPayload.HistorySummary, session)
	}
	jsonBytes, err := json.MarshalIndent(contextPayload, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling context JSON: %w", err)
	}
	fmt.Printf("🚀 Generating optimized AI dataset file from SQLite at: %s\n", outputPath)
	return os.WriteFile(outputPath, jsonBytes, 0644)
}
