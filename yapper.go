package yapper

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"slices"
	"time"

	"github.com/AleksaSvitlica/yapper/history"
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

type Pairings struct {
	data [][2]ID
}

// NewPairingsFromFile constructs and returns Pairings.
func NewPairingsFromFile(path string) (Pairings, error) {
	file, err := os.Open(path)
	if err != nil {
		return Pairings{}, fmt.Errorf("error opening file %s: %w", path, err)
	}

	data := new([][2]ID)
	err = json.NewDecoder(file).Decode(data)
	if err != nil {
		return Pairings{}, fmt.Errorf("error decoding Pairings: %w", err)
	}

	return Pairings{data: *data}, nil
}

// Export writes the pairings to the given writer, typically a file.
func (p *Pairings) Export(writer io.Writer) error {
	data, err := json.Marshal(p.data)
	if err != nil {
		return fmt.Errorf("error marshalling Pairings: %w", err)
	}

	if _, err = writer.Write(data); err != nil {
		return fmt.Errorf("error writing Pairings: %w", err)
	}
	return nil
}

func (p *Pairings) Add(id1, id2 ID) {
	p.data = append(p.data, [2]ID{id1, id2})
}

func (p *Pairings) All() iter.Seq2[ID, ID] {
	return func(yield func(ID, ID) bool) {
		for _, pair := range p.data {
			if !yield(pair[0], pair[1]) {
				return
			}
		}
	}
}

func GeneratePairings(config Config, hist *history.History, weeks int) ([]Pairings, error) {
	date := time.Now()
	var weeklyPairings []Pairings
	idToValidPairings := determineValidPairings(config)

	for range weeks {
		pairings := pairPeople(config, idToValidPairings, *hist, date)

		for id1, id2 := range pairings.All() {
			hist.AddMeeting(
				history.ID(id1),
				history.ID(id2),
				date,
			)
		}

		weeklyPairings = append(weeklyPairings, pairings)
		date = date.AddDate(0, 0, 7)
	}

	return weeklyPairings, nil
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
func pairPeople(conf Config, idToValidPairings map[ID][]ID, hist history.History, date time.Time) Pairings {
	pairings := Pairings{}
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
			pairings.Add(id, pair)
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
		if slices.Contains(validPairings, ID(prevID)) {
			possiblePairingsOrdered = append(possiblePairingsOrdered, ID(prevID))
		}
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
