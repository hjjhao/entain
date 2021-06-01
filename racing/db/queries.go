package db

const (
	racesList  = "raceslist"
	sportsList = "sportslist"
)

func getRaceQueries() map[string]string {
	return map[string]string{
		racesList: `
			SELECT 
				id, 
				meeting_id, 
				name, 
				number, 
				visible, 
				advertised_start_time 
			FROM races
		`,
	}
}

func getSportsQueries() map[string]string {
	return map[string]string{
		sportsList: `
			SELECT 
				id,
				name,
				athletics,
				location,
				following,
				visible,
				advertised_start_time 
			FROM sports
		`,
	}
}
