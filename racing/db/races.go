package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	_ "github.com/mattn/go-sqlite3"

	"git.neds.sh/matty/entain/racing/proto/racing"
)

// RacesRepo provides repository access to races.
type RacesRepo interface {
	// Init will initialise our races repository.
	Init() error

	// List will return a list of races.
	List(filter *racing.ListRacesRequestFilter) ([]*racing.Race, error)
	GetRaceById(id *int64) ([]*racing.Race, error)
}

type racesRepo struct {
	db   *sql.DB
	init sync.Once
}

// NewRacesRepo creates a new races repository.
func NewRacesRepo(db *sql.DB) RacesRepo {
	return &racesRepo{db: db}
}

// Init prepares the race repository dummy data.
func (r *racesRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy races.
		err = r.seed()
	})

	return err
}

func (r *racesRepo) List(filter *racing.ListRacesRequestFilter) ([]*racing.Race, error) {
	var (
		err   error
		query string
		args  []interface{}
	)

	query = getRaceQueries()[racesList]

	query, args = r.applyFilter(query, filter)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanRaces(rows)
}

func (r *racesRepo) applyFilter(query string, filter *racing.ListRacesRequestFilter) (string, []interface{}) {
	var (
		clauses        []string
		args           []interface{}
		orderDirection string = "DESC"
	)

	if filter == nil {
		return query, args
	}

	if len(filter.MeetingIds) > 0 {
		clauses = append(clauses, "meeting_id IN ("+strings.Repeat("?,", len(filter.MeetingIds)-1)+"?)")

		for _, meetingID := range filter.MeetingIds {
			args = append(args, meetingID)
		}
	}

	// filtering all the races with visible = 1
	if filter.Options != nil && filter.Options.VisibleOnly == true {
		clauses = append(clauses, "visible = ?")
		args = append(args, 1)
	}

	// filtering all the races with visible = 0
	if filter.Options != nil && filter.Options.VisibleOnly == false {
		clauses = append(clauses, "visible = ?")
		args = append(args, 0)
	}

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	// Add sort ability
	if filter.Options != nil && filter.Options.OrderBy != "" {
		if filter.Options.OrderDirection == "ASC" || filter.Options.OrderDirection == "DESC" {
			orderDirection = filter.Options.OrderDirection
		}
		query += fmt.Sprintf(" ORDER BY %v %v", filter.Options.OrderBy, orderDirection)
	}

	return query, args
}

func (r *racesRepo) GetRaceById(id *int64) ([]*racing.Race, error) {
	var (
		err   error
		query string
		args  []interface{}
	)

	query = getRaceQueries()[racesList]

	query, args = r.getSingleRaceFilter(query, id)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanRaces(rows)
}

//
func (r *racesRepo) getSingleRaceFilter(query string, id *int64) (string, []interface{}) {
	var (
		clauses []string
		args    []interface{}
	)

	if id == nil {
		return query, args
	}

	clauses = append(clauses, "id = ?")
	args = append(args, *id)

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	return query, args
}

func (m *racesRepo) scanRaces(
	rows *sql.Rows,
) ([]*racing.Race, error) {
	var races []*racing.Race

	for rows.Next() {
		var race racing.Race
		var advertisedStart time.Time

		if err := rows.Scan(&race.Id, &race.MeetingId, &race.Name, &race.Number, &race.Visible, &advertisedStart); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}

			return nil, err
		}

		// Add status field
		if advertisedStart.Before(time.Now()) {
			race.Status = "CLOSED"
		} else {
			race.Status = "OPEN"
		}

		ts, err := ptypes.TimestampProto(advertisedStart)
		if err != nil {
			return nil, err
		}

		race.AdvertisedStartTime = ts

		races = append(races, &race)
	}

	return races, nil
}
