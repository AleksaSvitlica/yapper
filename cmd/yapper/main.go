package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/AleksaSvitlica/yapper"
	"github.com/AleksaSvitlica/yapper/history"
)

const (
	exitCodeSuccess          = 0
	exitCodeError            = 1
	exitCodeInvalidArguments = 2
)

func main() {
	os.Exit(execute(os.Args[1:]))
}

func execute(args []string) int {
	cmd := flag.NewFlagSet("yapper", flag.ContinueOnError)
	pathToConfig := cmd.String("config", "", "Path to a yapper config file.")
	pathToHistory := cmd.String("history", "history.json", "Path to a yapper history file. The updated history will be written to this file as well.")
	weeksOfPairings := cmd.Int("weeks", 1, "Number of weeks of pairings to generate.")
	if err := cmd.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return exitCodeInvalidArguments
	}

	config, err := yapper.NewConfigFromFile(*pathToConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n", err)
		return exitCodeError
	}

	hist, err := getHistoryFromFile(*pathToHistory, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting history from file: %v\n", err)
		return exitCodeError
	}

	_, err = yapper.GeneratePairings(config, &hist, *weeksOfPairings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating pairings: %v\n", err)
		return exitCodeError
	}

	if err := writeHistoryToFile(hist, *pathToHistory); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing updated history to file: %s, %v\n", *pathToHistory, err)
		return exitCodeError
	}

	return exitCodeSuccess
}

// getHistoryFromFile will get the history from a file at the given path.
// If allowMissing is true then an empty history will be returned if the file does not exist.
func getHistoryFromFile(path string, allowMissing bool) (history.History, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if allowMissing {
			return history.History{}, nil
		}
		return history.History{}, fmt.Errorf("history file does not exist: %s, %w", path, err)
	} else if err != nil {
		return history.History{}, err
	}

	hist, err := history.NewHistoryFromFile(file)
	if err != nil {
		return history.History{}, err
	}

	if err := file.Close(); err != nil {
		return history.History{}, err
	}

	return hist, nil
}

func writeHistoryToFile(hist history.History, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating history output file: %s, %w", path, err)
	}

	if err := hist.Export(file); err != nil {
		return fmt.Errorf("error exporting history to file: %s, %w", path, err)
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}
