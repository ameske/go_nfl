package database

import (
	"log"
	"time"
)

type Weeks struct {
	Id        int64 `db:"id"`
	YearId    int64 `db:"year_id"`
	PvsId     int64 `db:"pvs_id"`
	Week      int   `db:"week"`
	WeekStart int64 `db:"week_start"`
}

func WeekId(year, week int) int64 {
	var weekId int64
	err := db.SelectOne(&weekId, "SELECT weeks.id FROM weeks JOIN years ON weeks.year_id = years.id WHERE years.year = $1 AND weeks.week = $2", year, week)
	if err != nil {
		log.Fatalf("WeekId: %s", err.Error())
	}

	return weekId
}

func CurrentWeek() (year, week int) {
	t := time.Now().Unix()
	year = time.Now().Year()
	err := db.SelectOne(&week, "SELECT MAX(week) FROM weeks JOIN years ON years.id = weeks.year_id WHERE year = $1 AND $2 > weeks.week_start", year, t)
	if err != nil {
		log.Fatalf("CurrentWeek: %s", err.Error())
	}

	return
}
