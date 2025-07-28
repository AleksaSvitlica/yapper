package yapper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/AleksaSvitlica/yapper/internal/history"
)

type Cadence string

const (
	CadenceOneWeek  Cadence = "one-week"
	CadenceTwoWeeks Cadence = "two-weeks"
)

type ID string

type Config struct {
	People []Person `json:"people"`
}

func (c Config) GetPerson(id ID) (Person, error) {
	index := slices.IndexFunc(c.People, func(p Person) bool {
		return p.ID == id
	})

	if index == -1 {
		return Person{}, fmt.Errorf("no person with ID %s", id)
	}

	return c.People[index], nil
}

func (c Config) validate() error {
	ids := make(map[ID]struct{})
	for _, person := range c.People {
		_, exists := ids[person.ID]
		if exists {
			return fmt.Errorf("ID is not unique: %s", person.ID)
		}
		ids[person.ID] = struct{}{}
	}
	return nil
}

func NewConfigFromFile(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("error opening file %s: %w", path, err)
	}

	config := new(Config)
	err = json.NewDecoder(file).Decode(config)
	if err != nil {
		return Config{}, fmt.Errorf("error decoding Config: %w", err)
	}

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	if err := file.Close(); err != nil {
		return Config{}, err
	}

	return *config, nil
}

type Person struct {
	ID       ID      `json:"id"`
	DenyList []ID    `json:"denyList"`
	Cadence  Cadence `json:"cadence"`
	Squad    string  `json:"squad"`
}

func GeneratePairings(config Config, hist *history.History, weeks int) error {
	date := time.Now()

	idToValidPairings := determineValidPairings(config)

	for i := range weeks {
		fmt.Printf("Week %d: %s\n", i, date.Format(time.DateOnly))
		pairings := pairPeople(config, idToValidPairings, *hist, date)
		fmt.Printf("\tPairings: %d\n", len(pairings))

		for _, pairing := range pairings {
			hist.AddMeeting(
				history.ID(pairing[0]),
				history.ID(pairing[1]),
				date,
			)

			fmt.Printf("\tPairing: %s and %s\n", pairing[0], pairing[1])
		}

		date = date.AddDate(0, 0, 7)
	}

	return nil
}

// determineValidPairings parses the people and their deny lists to determine the valid pairings for each person.
func determineValidPairings(config Config) map[ID][]ID {
	pairings := map[ID][]ID{}

	for _, person := range config.People {
		for _, potentialPair := range config.People {
			if person.ID == potentialPair.ID ||
				slices.Contains(person.DenyList, potentialPair.ID) || slices.Contains(potentialPair.DenyList, person.ID) {
				continue
			}

			if person.Squad != "" && person.Squad == potentialPair.Squad {
				continue
			}

			pairings[person.ID] = append(pairings[person.ID], potentialPair.ID)
		}
	}

	return pairings
}

// pairPeople based on their valid pairings.
// Preference is given to unmet people and then by longest time since last meeting.
func pairPeople(conf Config, idToValidPairings map[ID][]ID, hist history.History, date time.Time) [][2]ID {
	var pairings [][2]ID
	alreadyPaired := getIneligiblePeople(conf, idToValidPairings, date)

	for id, validPairings := range idToValidPairings {
		if slices.Contains(alreadyPaired, id) {
			continue
		}

		orderedPossiblePairings := getOrderedPossiblePairings(id, validPairings, hist)
		for _, pair := range orderedPossiblePairings {
			if slices.Contains(alreadyPaired, pair) {
				continue
			}
			pairings = append(pairings, [2]ID{id, pair})
			alreadyPaired = append(alreadyPaired, id)
			alreadyPaired = append(alreadyPaired, pair)
			break
		}
	}

	return pairings
}

// getOrderedPossiblePairings sorts the valid pairings based on the time since last meeting in descending order.
// Any possible pairings that have not been met will be placed in the front to ensure priority.
func getOrderedPossiblePairings(id ID, validPairings []ID, hist history.History) []ID {
	previousMeetingsOldestFirst := history.GetPeopleMetSortedByLastMeeting(hist, history.ID(id))
	unmetPeople := getPeopleNotMetBefore(validPairings, previousMeetingsOldestFirst)

	possiblePairingsOrdered := unmetPeople
	for _, prevID := range previousMeetingsOldestFirst {
		possiblePairingsOrdered = append(possiblePairingsOrdered, ID(prevID))
	}

	return possiblePairingsOrdered
}

func getPeopleNotMetBefore(validPairings []ID, previousPairings []history.ID) []ID {
	var unmetPeople []ID
	for _, id := range validPairings {
		if !slices.Contains(previousPairings, history.ID(id)) {
			unmetPeople = append(unmetPeople, id)
		}
	}

	return unmetPeople
}

// getIneligiblePeople returns the IDs of the people who cannot meet this week.
func getIneligiblePeople(conf Config, idToValidPairings map[ID][]ID, date time.Time) []ID {
	var ineligible []ID

	twoWeekValid := isValidWeekForTwoWeekCadence(date)

	for id := range idToValidPairings {
		person, err := conf.GetPerson(id)
		if err != nil {
			log.Fatalf("Cannot find %s in config", id)
		}

		switch person.Cadence {
		case CadenceOneWeek, "":
			continue
		case CadenceTwoWeeks:
			if !twoWeekValid {
				ineligible = append(ineligible, id)
			}
		default:
			log.Fatalf("Unexpected cadence: %s", person.Cadence)
		}
	}

	return ineligible
}

func isValidWeekForTwoWeekCadence(date time.Time) bool {
	_, week := date.ISOWeek()
	return (week % 2) == 0
}
