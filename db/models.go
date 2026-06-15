package db

type Exercise struct {
	ID           int64
	Name         string
	MuscleGroup  string
	IsUnilateral bool
}

type Serie struct {
	ID           int64
	SessionID    int64
	ExerciseID   int64
	SeriesNumber int
	Reps         int
	Weight       float64
	RPE          *int
}

