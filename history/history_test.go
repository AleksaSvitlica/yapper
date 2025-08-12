package history

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	mario  ID = "mario"
	luigi  ID = "luigi"
	peach  ID = "peach"
	bowser ID = "bowser"
)

func TestLatestMeetingsReturnsExpected(t *testing.T) {
	person1 := ID("person1")
	person2 := ID("person2")
	meetingTime := time.Now()

	hist := History{}
	hist.AddMeeting(person1, person2, meetingTime)

	// Confirm latest meetings for person1 with person2 show new time
	personToLastMeeting := hist.GetPersonToLastMeetingMap(person1)
	lastMeeting, exists := personToLastMeeting[person2]
	if !exists {
		t.Fatalf("Expected an entry for %s in: %v", person2, personToLastMeeting)
	}

	if lastMeeting != meetingTime {
		t.Errorf("Expected %s's last meeting with %s to be %v, got: %v", person1, person2, meetingTime, lastMeeting)
	}

	// Confirm latest meetings for person2 with person1 shows new time
	personToLastMeeting = hist.GetPersonToLastMeetingMap(person2)
	lastMeeting, exists = personToLastMeeting[person1]
	if !exists {
		t.Fatalf("Expected an entry for %s in: %v", person1, personToLastMeeting)
	}

	if lastMeeting != meetingTime {
		t.Errorf("Expected %s's last meeting with %s to be %v, got: %v", person2, person1, meetingTime, lastMeeting)
	}
}

func TestGetPeopleMetSortedByLastMeeting(t *testing.T) {
	person1 := ID("person1")
	person2 := ID("person2")
	person3 := ID("person3")
	person4 := ID("person4")

	now := time.Now()
	oneDayAgo := time.Now().AddDate(0, 0, -1)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)

	hist := History{}
	hist.AddMeeting(person1, person2, now)
	hist.AddMeeting(person1, person3, oneDayAgo)
	hist.AddMeeting(person1, person4, twoDaysAgo)

	expectedPeople := []ID{person4, person3, person2}
	sortedPeople := GetPeopleMetSortedByLastMeeting(hist, person1)

	if eq := reflect.DeepEqual(sortedPeople, expectedPeople); !eq {
		t.Errorf("Expected:\n%v\ngot:\n%v", expectedPeople, sortedPeople)
	}
}

func TestHistoryExportWritesExpectedData(t *testing.T) {
	hist := getExpectedHistory()

	expectedDataFile := "./testdata/expected_history.json"

	var writeBuffer bytes.Buffer

	if err := hist.Export(&writeBuffer); err != nil {
		t.Errorf("unexpected error from Export: %v", err)
	}

	actual := writeBuffer.String()
	expected, err := readExpectedData(t, expectedDataFile)
	if err != nil {
		t.Fatalf("error reading expected data from %s: %v", expectedDataFile, err)
	}
	if actual != expected {
		t.Errorf("\nexpected:\n%q\ngot:\n%q\n", expected, actual)
	}
}

func TestNewHistoryFromFileResultsInExpectedHistory(t *testing.T) {
	expectedHistory := getExpectedHistory()
	expectedDataFile := "./testdata/expected_history.json"

	file, err := os.Open(expectedDataFile)
	if err != nil {
		t.Errorf("error opening file %s: %v", expectedDataFile, err)
	}

	hist, err := NewHistoryFromFile(file)
	if err != nil {
		t.Errorf("unexpected error from NewHistoryFromFile: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Errorf("Error closing file: %v", err)
	}

	assertHistoriesEqual(t, expectedHistory, hist)
}

func getExpectedHistory() History {
	date1 := time.Date(2025, time.July, 20, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, time.June, 5, 0, 0, 0, 0, time.UTC)

	data := map[ID]map[ID]time.Time{
		mario: {
			luigi: date1,
			peach: date2,
		},
		luigi: {
			mario:  date1,
			bowser: date2,
		},
		peach: {
			mario: date2,
		},
		bowser: {
			luigi: date2,
		},
	}

	return History{data: data}
}

func assertHistoriesEqual(t *testing.T, expected, actual History) {
	t.Helper()
	if eq := reflect.DeepEqual(actual, expected); !eq {
		t.Errorf("\nexpected:\n%s\ngot:\n%s\n", stringFormatHistory(expected), stringFormatHistory(actual))
	}
}

func stringFormatHistory(history History) string {
	var sb strings.Builder

	for id, idToTimes := range history.data {
		sb.WriteString(fmt.Sprintf("\t%v: %v\n", id, idToTimes))
	}

	return sb.String()
}

func readExpectedData(t *testing.T, path string) (string, error) {
	t.Helper()
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(fileBytes)), nil
}
