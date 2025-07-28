package history

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"
)

type ID string

// History keeps track of which people have met and when their last meeting was.
type History struct {
	data map[ID]map[ID]time.Time
}

// NewHistoryFromFile attempts to unmarshal the data from the given reader and return a History.
func NewHistoryFromFile(reader io.Reader) (History, error) {
	history := History{}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&history.data); err != nil {
		return history, fmt.Errorf("error decoding history: %w", err)
	}
	return history, nil
}

// AddMeeting updates the meeting time for the given people.
func (h *History) AddMeeting(person1 ID, person2 ID, meetingTime time.Time) {
	if h.data == nil {
		h.data = make(map[ID]map[ID]time.Time)
	}

	h.addMeetingToPersonsHistory(person1, person2, meetingTime)
	h.addMeetingToPersonsHistory(person2, person1, meetingTime)
}

// GetPersonToLastMeetingMap returns a map of the people they have met and the time of that meeting.
func (h *History) GetPersonToLastMeetingMap(person ID) map[ID]time.Time {
	personHistory, exists := h.data[person]
	if !exists {
		return nil
	}

	return personHistory
}

// Export writes the history data to the given writer, typically a file.
func (h *History) Export(writer io.Writer) error {
	data, err := json.Marshal(h.data)
	if err != nil {
		return fmt.Errorf("error marshalling history: %w", err)
	}

	if _, err = writer.Write(data); err != nil {
		return fmt.Errorf("error writing history: %w", err)
	}
	return nil
}

func (h *History) addMeetingToPersonsHistory(person ID, otherPerson ID, meetingTime time.Time) {
	personHistory, exists := h.data[person]
	if !exists {
		personHistory = make(map[ID]time.Time)
	}

	personHistory[otherPerson] = meetingTime
	h.data[person] = personHistory
}

// GetPeopleMetSortedByLastMeeting returns a slice of people they have met in decreasing time since last meeting.
func GetPeopleMetSortedByLastMeeting(hist History, person ID) []ID {
	peopleToTime := hist.GetPersonToLastMeetingMap(person)
	var sortedPeople []ID

	for p, meetingTime := range peopleToTime {
		index := 0
		for _, sortedPerson := range sortedPeople {
			if meetingTime.Before(peopleToTime[sortedPerson]) {
				break
			}
			index++
		}

		sortedPeople = slices.Insert(sortedPeople, index, p)
	}

	return sortedPeople
}
