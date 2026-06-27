package exporter

import (
	"database/sql"
	"encoding/json"
	"os"
)

type IASession struct {
	Date     string       `json:"date"`
	Cardio   *IACardio    `json:"cardio,omitempty"`
	Workouts []IAWorkout  `json:"workouts,omitempty"`
}
type IACardio struct {
	Duration float64 `json:"duration_min"`
	Distance float64 `json:"distance_km"`
	AvgSpeed float64 `json:"avg_speed_kmh"`
	MaxSpeed float64 `json:"max_speed_kmh"`
	Calories float64 `json:"calories_kcal"`
	MaxHR    int     `json:"max_hr_bpm"`
}
type IAWorkout struct {
	Exercise string   `json:"exercise"`
	Series   []IASet  `json:"series"`
}
type IASet struct {
	SetIndex int     `json:"set"`
	Reps     int     `json:"reps"`
	Weight   float64 `json:"weight_kg"`
}
type IAUserData struct {
	Birthdate     string  `json:"birthdate"`
	Gender        string  `json:"gender"`
	Height        float64 `json:"height_m"`
	InitialWeight float64 `json:"initial_weight_kg"`
	CurrentWeight float64 `json:"current_weight_kg"`
	WeightDate    string  `json:"weight_date"`
	HealthContext string  `json:"health_context"`
}
type IAExportPayload struct {
	User    IAUserData  `json:"user_metadata"`
	History []IASession `json:"training_history"`
}

func ExportToIAJson(db *sql.DB, outputPath string) error {
	payload := IAExportPayload{
		History: []IASession{},
	}
	userQuery := `SELECT birth_date, gender, height_m, initial_weight_kg, current_weight_kg, last_weight_date, health_context 
	              FROM user_metadata ORDER BY id DESC LIMIT 1`
	err := db.QueryRow(userQuery).Scan(
		&payload.User.Birthdate, &payload.User.Gender, &payload.User.Height,
		&payload.User.InitialWeight, &payload.User.CurrentWeight, &payload.User.WeightDate,
		&payload.User.HealthContext,
	)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	sessionRows, err := db.Query(`SELECT id, date_of_exercise FROM sessions ORDER BY date_of_exercise ASC`)
	if err != nil {
		return err
	}
	defer sessionRows.Close()
	for sessionRows.Next() {
		var sID int
		var sDate string
		if err := sessionRows.Scan(&sID, &sDate); err != nil {
			return err
		}
		sessionObj := IASession{Date: sDate}
		cardioQuery := `SELECT duration_minutes, distance_km, avg_speed, max_speed, calories, max_heart_rate 
		                FROM cardio_sessions WHERE session_id = ?`
		var cDur, cDist, cAvgS, cMaxS, cCal float64
		var cMaxHR sql.NullInt32
		err = db.QueryRow(cardioQuery, sID).Scan(&cDur, &cDist, &cAvgS, &cMaxS, &cCal, &cMaxHR)
		if err == nil {
			sessionObj.Cardio = &IACardio{
				Duration: cDur, Distance: cDist, AvgSpeed: cAvgS,
				MaxSpeed: cMaxS, Calories: cCal, MaxHR: int(cMaxHR.Int32),
			}
		}
		workoutRows, err := db.Query(`
			SELECT DISTINCT e.id, e.name 
			FROM series s
			JOIN exercises e ON s.exercise_id = e.id
			WHERE s.session_id = ?`, sID)
		if err != nil {
			return err
		}
		for workoutRows.Next() {
			var eID int
			var eName string
			if err := workoutRows.Scan(&eID, &eName); err != nil {
				workoutRows.Close()
				return err
			}
			workoutObj := IAWorkout{Exercise: eName, Series: []IASet{}}
			setRows, err := db.Query(`
				SELECT series_number, reps, weight 
				FROM series 
				WHERE session_id = ? AND exercise_id = ? 
				ORDER BY series_number ASC`, sID, eID)
			if err != nil {
				workoutRows.Close()
				return err
			}
			for setRows.Next() {
				var sIdx, sReps int
				var sWgt float64
				if err := setRows.Scan(&sIdx, &sReps, &sWgt); err != nil {
					setRows.Close()
					workoutRows.Close()
					return err
				}
				workoutObj.Series = append(workoutObj.Series, IASet{
					SetIndex: sIdx,
					Reps:     sReps,
					Weight:   sWgt,
				})
			}
			setRows.Close()
			if len(workoutObj.Series) > 0 {
				sessionObj.Workouts = append(sessionObj.Workouts, workoutObj)
			}
		}
		workoutRows.Close()
		payload.History = append(payload.History, sessionObj)
	}
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, jsonData, 0644)
}
