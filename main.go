package main

import (
	"fmt"
	"log"
	"workout-tracker-go/db"
	"workout-tracker-go/parser"
)

func main() {
	fmt.Println("🏋️ Initializing Massive Workout Ingestion...")
	database := db.InitDB("training.db")
	defer database.Close()
	healthContext := "Informático, trabajo todo el día sentado. Promedio de sueño: 6h. " +
		"Historial familiar: Infartos (padre y tío paterno fallecidos), Cáncer (hermana paterna, tía paterna, tía materna y prima materna). " +
		"Casos de Alzheimer y Parkinson en familia materna. " +
		"Objetivo: Mejorar resistencia cardiovascular y aumentar masa muscular con énfasis en las piernas."
	err := db.SaveUserMetadata(
		database,
		"1985-06-27",
		"hombre",
		1.70,
		83.0,
		81.0,
		"2026-06-05",
		healthContext,
	)
	if err != nil {
		log.Fatalf("❌ Error saving user metadata: %v", err)
	}
	markdownPath := "Registro de ejercicios.md"
	fmt.Printf("📖 Parsing history file: %s\n", markdownPath)
	sessions, err := parser.ParseFullMarkdown(markdownPath)
	if err != nil {
		log.Fatalf("❌ Error reading markdown file: %v", err)
	}
	fmt.Printf("🎯 Found %d training sessions in the file. Migrating to SQLite...\n\n", len(sessions))
	for _, session := range sessions {
		if session.Date == "Unknown" || session.Date == "" {
			continue
		}
		sessionID, err := db.GetOrCreateSession(database, session.Date)
		if err != nil {
			log.Printf("❌ Error creating session for date %s: %v", session.Date, err)
			continue
		}
		if session.Cardio != nil && (session.Cardio.Duration > 0 || session.Cardio.Distance > 0) {
			err = db.SaveCardio(database, sessionID,
				session.Cardio.Duration,
				session.Cardio.Distance,
				session.Cardio.AvgSpeed,
				session.Cardio.MaxSpeed,
				session.Cardio.Calories,
				session.Cardio.MaxHR,
			)
			if err != nil {
				log.Printf("⚠️ Error saving cardio for day %s: %v", session.Date, err)
			} else {
				fmt.Printf("🏃 [%s] Cardio synced: %.2f km in %.1f min\n", session.Date, session.Cardio.Distance, session.Cardio.Duration)
			}
		}
		for _, workout := range session.Workouts {
			dummyLine := workout.ExerciseName + " " + workout.RawSeries
			exerciseName, series, err := parser.ParseLine(dummyLine)
			if err != nil {
				continue
			}
			exerciseID, err := db.GetOrCreateExercise(database, exerciseName)
			if err != nil {
				log.Printf("❌ Error managing exercise %s: %v", exerciseName, err)
				continue
			}
			err = db.SaveSeries(database, sessionID, exerciseID, series)
			if err != nil {
				log.Printf("❌ Error saving series for %s on %s: %v", exerciseName, session.Date, err)
			} else {
				fmt.Printf("🏋️ [%s] %s -> %d series guardadas.\n", session.Date, exerciseName, len(series))
			}
		}
	}
	fmt.Println("\n🚀 Historic migration complete! Core relational data and cardio metrics are fully synchronized.")
}
