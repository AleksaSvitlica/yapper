package yapper

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/AleksaSvitlica/yapper/internal/history"
)

const validConfigName = "validConfig.json"

func TestNewConfigFromFileReturnsErrorIfFileDoesNotExist(t *testing.T) {
	configPath := getPathToConfig(t, "non-existent-config.json")
	if _, err := NewConfigFromFile(configPath); err == nil {
		t.Errorf("Expected error due to file not existing: %s", configPath)
	}
}

func TestNewConfigFromFileReturnsExpectedConfig(t *testing.T) {
	configPath := getPathToConfig(t, validConfigName)
	expectedPeople := []Person{
		{ID: "Mario", DenyList: []ID{"Wario", "Bowser"}, Squad: "bros"},
		{ID: "Luigi", DenyList: []ID{"Waluigi", "Bowser"}, Squad: "bros"},
		{ID: "Wario", DenyList: []ID{"Mario"}},
		{ID: "Waluigi", DenyList: []ID{"Luigi"}},
		{ID: "Toad"},
		{ID: "Yoshi"},
		{ID: "Peach", DenyList: []ID{"Bowser"}},
		{ID: "Bowser", Squad: "koopas"},
		{ID: "Bowser Jr", Squad: "koopas"},
		{ID: "Shy Guy", DenyList: []ID{"Mario", "Luigi", "Peach"}, Cadence: CadenceTwoWeeks},
		{ID: "Monty Mole", Cadence: CadenceTwoWeeks},
		{ID: "Koopa Troopa", Squad: "koopas"},
	}
	expectedConfig := Config{People: expectedPeople}
	config, err := NewConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if eq := reflect.DeepEqual(config, expectedConfig); !eq {
		t.Errorf("Expected:\n%v\nGot:\n%v", expectedConfig, config)
	}
}

func TestNewConfigFromReturnsErrorIfNotAllIDsAreUnique(t *testing.T) {
	configPath := getPathToConfig(t, "nonUniqueIDsConfig.json")
	if _, err := NewConfigFromFile(configPath); err == nil {
		t.Errorf("Expected error due to non-unique ID: %s", configPath)
	}
}

func TestConfigGetPersonReturnsExpectedPerson(t *testing.T) {
	config := getConfigFromFile(t, validConfigName)
	id := ID("Mario")
	expectedPerson := Person{ID: id, DenyList: []ID{"Wario", "Bowser"}, Squad: "bros"}

	person, err := config.GetPerson(id)
	if err != nil {
		t.Fatalf("Unexpected error from Config.GetPerson, %s: %v", id, err)
	}

	if eq := reflect.DeepEqual(person, expectedPerson); !eq {
		t.Errorf("Expected:\n%v\nGot:%v", expectedPerson, person)
	}
}

func TestConfigGetPersonReturnsErrorWhenPersonDoesNotExist(t *testing.T) {
	config := getConfigFromFile(t, validConfigName)
	id := ID("sonic")
	if _, err := config.GetPerson(id); err == nil {
		t.Errorf("Expected error due to %s not in config", id)
	}
}

func TestDetermineValidPairingsReturnsCorrectPairings(t *testing.T) {
	config := getConfigFromFile(t, validConfigName)
	expected := getValidPairsForConfig()
	validPairings := determineValidPairings(config)
	diffPairings(t, validPairings, expected)
}

func TestPairPeopleDoesNotCreateInvalidPairs(t *testing.T) {
	config := getConfigFromFile(t, validConfigName)
	validPairs := getValidPairsForConfig()
	hist := history.History{}
	date := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)

	for _, pair := range pairPeople(config, validPairs, hist, date) {
		checkPairing := func(t *testing.T, person1 ID, person2 ID, validPairs map[ID][]ID) {
			t.Helper()
			validFor1, exists := validPairs[person1]
			if !exists {
				t.Fatalf("Got unexpected person: %s", person1)
			}

			if !slices.Contains(validFor1, person2) {
				t.Fatalf("%s cannot be paired with %s", person1, person2)
			}
		}
		checkPairing(t, pair[0], pair[1], validPairs)
		checkPairing(t, pair[1], pair[0], validPairs)
	}
}

func TestAverageTimeSinceMeetingIncreasesOrIsGreaterThanMinimum(t *testing.T) {
	minimumAvgDaysSinceMeeting := 14.0
	weeksOfPairings := 30
	date := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)

	config := getConfigFromFile(t, validConfigName)
	hist := history.History{}
	validPairs := getValidPairsForConfig()
	allIDs := getAllIDs(t, config)

	lastAvg := -0.1
	for range weeksOfPairings {
		pairings := pairPeople(config, validPairs, hist, date)
		for _, pairing := range pairings {
			hist.AddMeeting(
				history.ID(pairing[0]),
				history.ID(pairing[1]),
				date,
			)
			t.Logf("Pair: %v", pairing)
		}

		avgDays := calculateAverageDaysSinceMeeting(t, date, allIDs, hist)
		t.Logf("Average days since meeting: %f", avgDays)

		if avgDays <= lastAvg && avgDays < minimumAvgDaysSinceMeeting {
			t.Errorf(
				"Average days on %v is below minimum (%f) and did not increase: %f <= %f",
				date,
				minimumAvgDaysSinceMeeting,
				avgDays,
				lastAvg,
			)
		}

		date = date.AddDate(0, 0, 7)
	}
}

func TestPeopleHaveMetAllEligiblePairs(t *testing.T) {
	date := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)

	config := getConfigFromFile(t, validConfigName)
	hist := history.History{}
	validPairs := getValidPairsForConfig()

	maxCadenceWeeks := 2
	weeksOfPairings := len(validPairs) * maxCadenceWeeks * 2

	for i := range weeksOfPairings {
		t.Logf("Week %d", i)
		pairings := pairPeople(config, validPairs, hist, date)

		for _, pairing := range pairings {
			hist.AddMeeting(
				history.ID(pairing[0]),
				history.ID(pairing[1]),
				date,
			)
			t.Logf("Pair: %v", pairing)
		}

		date = date.AddDate(0, 0, 7)
	}

	for person, validPairings := range validPairs {
		personToLastMeeting := hist.GetPersonToLastMeetingMap(history.ID(person))
		var peopleMet []ID
		for id := range personToLastMeeting {
			peopleMet = append(peopleMet, ID(id))
		}

		slices.Sort(validPairings)
		slices.Sort(peopleMet)

		if eq := reflect.DeepEqual(peopleMet, validPairings); !eq {
			t.Errorf("%s can meet with:\n%v\nbut only met:\n%v", person, validPairings, peopleMet)
		}
	}
}

func TestPeopleOnTwoWeekCadenceOnlyGetPairedEveryTwoWeeks(t *testing.T) {
	date := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)

	config := getConfigFromFile(t, validConfigName)
	hist := history.History{}
	validPairs := getValidPairsForConfig()

	weeksOfPairings := len(validPairs) + 4

	for i := range weeksOfPairings {
		t.Logf("Week %d", i)
		pairings := pairPeople(config, validPairs, hist, date)

		for _, pairing := range pairings {
			checkEligibleToMeetThisWeek(t, config, pairing[0], date)
			checkEligibleToMeetThisWeek(t, config, pairing[1], date)

			hist.AddMeeting(
				history.ID(pairing[0]),
				history.ID(pairing[1]),
				date,
			)
		}

		date = date.AddDate(0, 0, 7)
	}
}

func calculateAverageDaysSinceMeeting(t *testing.T, date time.Time, ids []ID, hist history.History) float64 {
	t.Helper()

	totalMeetings := 0
	var totalTime time.Duration

	for _, id := range ids {
		peopleToMeetingTimes := hist.GetPersonToLastMeetingMap(history.ID(id))
		for _, meetingTime := range peopleToMeetingTimes {
			totalTime += date.Sub(meetingTime)
			totalMeetings++
		}
	}

	if totalMeetings == 0 {
		return 0
	}

	return totalTime.Hours() / float64(totalMeetings*24)
}

func getConfigFromFile(t *testing.T, filename string) Config {
	t.Helper()

	configPath := getPathToConfig(t, filename)
	config, err := NewConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Error from NewConfigFromFile, %s: %v", configPath, err)
	}

	return config
}

func getPathToConfig(t *testing.T, filename string) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	return filepath.Join(dir, "testdata", filename)
}

func getAllIDs(t *testing.T, config Config) []ID {
	t.Helper()

	var allIDs []ID
	for _, person := range config.People {
		allIDs = append(allIDs, person.ID)
	}

	return allIDs
}

func diffPairings(t *testing.T, actual map[ID][]ID, expected map[ID][]ID) {
	t.Helper()

	var missing []ID
	var unexpected []ID
	var actualIDs []ID

	for id := range actual {
		actualIDs = append(actualIDs, id)
	}

	for id, expectedPairings := range expected {
		pairings, exists := actual[id]
		if exists {
			var pairingIDs []ID
			pairingIDs = append(pairingIDs, pairings...)

			slices.Sort(expectedPairings)
			slices.Sort(pairingIDs)

			if eq := reflect.DeepEqual(pairingIDs, expectedPairings); !eq {
				t.Errorf("For %s, expected pairings: %v\ngot:%v", id, expectedPairings, pairingIDs)
			}
		} else {
			missing = append(missing, id)
		}
	}

	for _, id := range actualIDs {
		if _, exists := expected[id]; !exists {
			unexpected = append(unexpected, id)
		}
	}

	if len(unexpected) != 0 {
		t.Errorf("Unexpected people with pairings: %v", unexpected)
	}

	if len(missing) != 0 {
		t.Errorf("People that were expected to have pairings but are missing: %v", missing)
	}
}

func checkEligibleToMeetThisWeek(t *testing.T, config Config, id ID, date time.Time) {
	t.Helper()

	twoWeekValid := isValidWeekForTwoWeekCadence(date)

	person, err := config.GetPerson(id)
	if err != nil {
		t.Fatalf("Could not retrieve %s from config: %v", id, err)
	}

	if person.Cadence == CadenceTwoWeeks && !twoWeekValid {
		t.Errorf("%s cannot meeting this week due to cadence: %s", id, person.Cadence)
	}
}

func getValidPairsForConfig() map[ID][]ID {
	return map[ID][]ID{
		"Mario":        {"Waluigi", "Toad", "Yoshi", "Peach", "Bowser Jr", "Monty Mole", "Koopa Troopa"},
		"Luigi":        {"Wario", "Toad", "Yoshi", "Peach", "Bowser Jr", "Monty Mole", "Koopa Troopa"},
		"Wario":        {"Luigi", "Waluigi", "Toad", "Yoshi", "Peach", "Bowser", "Bowser Jr", "Shy Guy", "Monty Mole", "Koopa Troopa"},
		"Waluigi":      {"Mario", "Wario", "Toad", "Yoshi", "Peach", "Bowser", "Bowser Jr", "Shy Guy", "Monty Mole", "Koopa Troopa"},
		"Toad":         {"Mario", "Luigi", "Wario", "Waluigi", "Yoshi", "Peach", "Bowser", "Bowser Jr", "Shy Guy", "Monty Mole", "Koopa Troopa"},
		"Yoshi":        {"Mario", "Luigi", "Wario", "Waluigi", "Toad", "Peach", "Bowser", "Bowser Jr", "Shy Guy", "Monty Mole", "Koopa Troopa"},
		"Peach":        {"Mario", "Luigi", "Wario", "Waluigi", "Toad", "Yoshi", "Bowser Jr", "Monty Mole", "Koopa Troopa"},
		"Bowser":       {"Wario", "Waluigi", "Toad", "Yoshi", "Shy Guy", "Monty Mole"},
		"Bowser Jr":    {"Mario", "Luigi", "Wario", "Waluigi", "Yoshi", "Peach", "Shy Guy", "Toad", "Monty Mole"},
		"Shy Guy":      {"Wario", "Waluigi", "Yoshi", "Bowser", "Bowser Jr", "Toad", "Monty Mole", "Koopa Troopa"},
		"Monty Mole":   {"Mario", "Luigi", "Wario", "Waluigi", "Yoshi", "Peach", "Bowser", "Bowser Jr", "Shy Guy", "Toad", "Koopa Troopa"},
		"Koopa Troopa": {"Mario", "Luigi", "Wario", "Waluigi", "Yoshi", "Peach", "Shy Guy", "Toad", "Monty Mole"},
	}
}
