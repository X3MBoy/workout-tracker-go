package db

import (
	"database/sql"
	"log"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("❌ Error opening database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("❌ Can't connect to SQLite: %v", err)
	}
	crearTablas(db)
	return db
}

func crearTablas(db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS exercises (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		muscle_group TEXT,
		is_unilateral INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date_of_exercise TEXT NOT NULL UNIQUE,
		comments TEXT
	);

	CREATE TABLE IF NOT EXISTS series (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		exercise_id INTEGER NOT NULL,
		series_number INTEGER NOT NULL,
		reps INTEGER NOT NULL,
		weight REAL NOT NULL,
		rpe INTEGER,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
		FOREIGN KEY (exercise_id) REFERENCES exercises(id) ON DELETE RESTRICT
	);

	CREATE TABLE IF NOT EXISTS cardio_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL UNIQUE,
		duration_minutes REAL,
		distance_km REAL,
		avg_speed REAL,
		max_speed REAL,
		calories REAL,
		max_heart_rate INTEGER,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS user_metadata (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        birth_date TEXT,
        gender TEXT,
        height_m REAL,
        initial_weight_kg REAL,
        current_weight_kg REAL,
        last_weight_date TEXT,
        health_context TEXT
    );
	`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatalf("❌ Error creating schema: %v", err)
	}
}

func GetOrCreateExercise(db *sql.DB, name string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM exercises WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO exercises (name, is_unilateral) VALUES (?, 0)", name)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func GetOrCreateSession(db *sql.DB, dateStr string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM sessions WHERE date_of_exercise = ?", dateStr).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO sessions (date_of_exercise) VALUES (?)", dateStr)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func SaveSeries(db *sql.DB, sessionID int64, exerciseID int64, series []Serie) error {
	_, err := db.Exec("DELETE FROM series WHERE session_id = ? AND exercise_id = ?", sessionID, exerciseID)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO series (session_id, exercise_id, series_number, reps, weight, rpe)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range series {
		_, err = stmt.Exec(sessionID, exerciseID, s.SeriesNumber, s.Reps, s.Weight, s.RPE)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func SaveCardio(database *sql.DB, sessionID int64, duration, distance, avgSpeed, maxSpeed, calories float64, maxHR int) error {
	_, err := database.Exec(`
		INSERT INTO cardio_sessions (session_id, duration_minutes, distance_km, avg_speed, max_speed, calories, max_heart_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			duration_minutes=excluded.duration_minutes,
			distance_km=excluded.distance_km,
			avg_speed=excluded.avg_speed,
			max_speed=excluded.max_speed,
			calories=excluded.calories,
			max_heart_rate=excluded.max_heart_rate
	`, sessionID, duration, distance, avgSpeed, maxSpeed, calories, maxHR)
	return err
}

func SaveUserMetadata(database *sql.DB, birthDate, gender string, height, initialWeight, currentWeight float64, lastWeightDate, healthContext string) error {
	_, err := database.Exec("DELETE FROM user_metadata")
	if err != nil {
		return err
	}
	query := `
    INSERT INTO user_metadata (
        birth_date, gender, height_m, initial_weight_kg, current_weight_kg, last_weight_date, health_context
    ) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = database.Exec(query, birthDate, gender, height, initialWeight, currentWeight, lastWeightDate, healthContext)
	return err
}
