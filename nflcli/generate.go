package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"text/tabwriter"

	"github.com/ameske/nfl-pickem/database"
)

func Generate(args []string) {
	if len(args) == 0 {
		GenerateHelp()
		return
	}

	switch args[0] {
	case "html":
		GenerateResultsHTML(args[1:])
	case "picks":
		GenerateSeasonPicks(args[1:])
	case "help":
		GenerateHelp()
	default:
		GenerateHelp()
	}

}

func GenerateHelp() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)

	fmt.Fprintln(w, "nfl-generate creates static assets or inits database tables")
	fmt.Fprintln(w, "\nAvailable commands:")
	fmt.Fprintln(w, "\thtml\t Generate results HTML for a given week")
	fmt.Fprintln(w, "\tpicks\t Generate empty pick rows for a sesason")
	fmt.Fprintf(w, "\n")

	err := w.Flush()
	if err != nil {
		log.Fatal(err)
	}
}

type ResultsTemplateData struct {
	Users   []database.Users
	Rows    []ResultsTableRow
	Totals  []int
	Title   string
	Content string
	End     string
}

type ResultsTableRow struct {
	Matchup string
	Picks   []UserPicks
}

type UserPicks struct {
	Pick   string
	Points int
	Status PickStatus
}

type PickStatus int

const (
	Correct PickStatus = iota
	Incorrect
	Pending
)

// GenerateResultsHTML creates an HTML file based on a template that displays the results for a given week.
func GenerateResultsHTML(args []string) {
	var year, week int

	f := flag.NewFlagSet("grade", flag.ExitOnError)
	f.IntVar(&year, "year", -1, "Year")
	f.IntVar(&week, "week", -1, "Week")

	err := f.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	if week == -1 || year == -1 {
		year, week = database.CurrentWeek()
	}

	teams := database.TeamAbbreviationMap()
	users := database.AllUsers()
	games := database.WeeklyGames(year, week)

	picks := make([][]*database.Picks, len(users))
	for i, u := range users {
		picks[i] = database.WeeklyPicksYearWeek(u.Email, year, week)
		reorderPicks(games, picks[i])
	}

	// Build each row of the table, where each row represents one game and all of the user's picks for that game
	rows := make([]ResultsTableRow, len(games))
	for i, g := range games {
		tr := ResultsTableRow{}

		tr.Matchup = fmt.Sprintf("%s/%s", teams[g.AwayId], teams[g.HomeId])

		tr.Picks = make([]UserPicks, len(users))
		for j, p := range picks {
			switch p[i].Selection {
			case 1:
				tr.Picks[j].Pick = fmt.Sprintf("%s", teams[games[i].AwayId])
			case 2:
				tr.Picks[j].Pick = fmt.Sprintf("%s", teams[games[i].HomeId])
			}
			tr.Picks[j].Points = p[i].Points

			if games[i].HomeScore == -1 && games[i].AwayScore == -1 {
				tr.Picks[j].Status = Pending
			} else if p[i].Correct {
				tr.Picks[j].Status = Correct
			} else {
				tr.Picks[j].Status = Incorrect
			}
		}
		rows[i] = tr
	}

	// Get the user's total for the week
	totals := make([]int, len(users))
	for i, user := range picks { //each user
		total := 0
		for _, p := range user { //each pick for the user
			if p.Correct {
				total += p.Points
			}
		}
		totals[i] = total
	}

	data := ResultsTemplateData{}
	data.Users = users
	data.Rows = rows
	data.Totals = totals
	data.Title = fmt.Sprintf("{{define \"title\"}}%d - Week %d Results{{end}}", year, week)
	data.Content = "{{define \"content\"}}"
	data.End = "{{end}}"

	t := template.New("results.html")
	t = template.Must(t.ParseFiles("/opt/ameske/gonfl/templates/results.html"))

	weekResults, err := os.Create(fmt.Sprintf("%d-Week%d-Results.html", year, week))
	if err != nil {
		log.Fatalf("CreatingFile: %s", err.Error())
	}
	defer weekResults.Close()

	err = t.Execute(weekResults, data)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

}

// Uses a stupid naive and slow algorithm to make sure that the picks line up with the games down the side.
// Fuck you postgres.
func reorderPicks(gamesOrder []database.Games, picks []*database.Picks) {
	for i, g := range gamesOrder {
		if picks[i].GameId != g.Id {
			// if it doesn't match go find the one that should be here and swap them
			for j := i; j < len(picks); j++ {
				if picks[j].GameId == g.Id {
					picks[i], picks[j] = picks[j], picks[i]
					break
				}
			}

		}
	}
}

type GamesWeeksJoin struct {
	database.Games
	database.Weeks
}

// SeasonPicks generates empty pick rows for a year's games. The games must be
// loaded into the database before this can be called.
func GenerateSeasonPicks(args []string) {
	var year int

	f := flag.NewFlagSet("generateSeasonPicks", flag.ExitOnError)
	f.IntVar(&year, "year", -1, "Year")

	err := f.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	if year == -1 {
		log.Fatalf("Year is a required argument")
	}

	database.CreateSeasonPicks(year)
}
