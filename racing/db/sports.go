package db

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"git.neds.sh/matty/entain/racing/proto/sports"
	"github.com/golang/protobuf/ptypes"
	_ "github.com/mattn/go-sqlite3"
)

// EventsRepo provides repository access to events.
type SportsRepo interface {
	// Init will initialise our events repository.
	Init() error

	// List will return a list of events.
	List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error)
}

type sportsRepo struct {
	db   *sql.DB
	init sync.Once
}

// NewEventsRepo creates a new events repository.
func NewSportsRepo(db *sql.DB) SportsRepo {
	return &sportsRepo{db: db}
}

// Init prepares the event repository dummy data.
func (r *sportsRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy events.
		// err = r.seed()
	})

	return err
}

func (r *sportsRepo) List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error) {
	var (
		err   error
		query string
		args  []interface{}
	)

	query = getSportsQueries()[sportsList]

	query, args = r.applyFilter(query, filter)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanEvents(rows)
}

func (r *sportsRepo) applyFilter(query string, filter *sports.ListEventsRequestFilter) (string, []interface{}) {
	var (
		clauses []string
		args    []interface{}
	)

	if filter == nil {
		return query, args
	}

	if len(filter.Ids) > 0 {
		clauses = append(clauses, "id IN ("+strings.Repeat("?,", len(filter.Ids)-1)+"?)")

		for _, id := range filter.Ids {
			args = append(args, id)
		}
	}

	// filtering all the events with visible = 1
	if filter.Options != nil && filter.Options.VisibleOnly == true {
		clauses = append(clauses, "visible = ?")
		args = append(args, 1)
	}

	// filtering all the events with visible = 0
	if filter.Options != nil && filter.Options.VisibleOnly == false {
		clauses = append(clauses, "visible = ?")
		args = append(args, 0)
	}

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	return query, args
}

func (m *sportsRepo) scanEvents(
	rows *sql.Rows,
) ([]*sports.Event, error) {
	var events []*sports.Event

	for rows.Next() {
		var event sports.Event
		var advertisedStart time.Time
		err := rows.Scan(&event.Id, &event.Name, &event.Athletics, &event.Location, &event.Following, &event.Visible, &advertisedStart)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}

		if &advertisedStart != nil {
			ts, err := ptypes.TimestampProto(advertisedStart)
			if err != nil {
				return nil, err
			}

			event.AdvertisedStartTime = ts
		}

		events = append(events, &event)
	}

	return events, nil
}
