# Yapper

Yapper is a tool for generating pairings for one-on-one meetings within a team.

## Features
- Generate each week or any number of weeks at a time.
- Individuals can opt-in to a 1 week or 2 week cadence for meetings.
- Deny lists for people you already meet with.
  - Can be individuals and/or squads.

## Usage
The tool depends on Golang.

Generating pairings for the example config can be done with:
```sh
go run ./cmd/yapper -config testdata/validConfig.json
```

A JSON file is used to track the history of the last meeting time between people. By default the tool reads and writes to this data to `history.json`. An alternative path can be used:
```sh
go run ./cmd/yapper -config testdata/validConfig.json -history path-to-history.json
```

Instead of generating new pairings every week it is also possible to generate multiple weeks of pairings at a time.
```sh
go run ./cmd/yapper -config testdata/validConfig.json -weeks 5
```

## Configuration file
The configuration file defines the people and their meeting preferences. See the [test configuration](testdata/validConfig.json) for a comprehensive example.

A person at minimum requires an ID:
```json
{
	"id": "Yoshi"
}
```

They can define people not to pair with if they already meet with them:
```json
{
	"id": "Waluigi",
	"denyList": ["Luigi"]
}
```

The deny list can be supplemented or replaced with a squad. Any people with a matching squad will not be paired.
```json
{
	"id": "Mario",
	"denyList": ["Wario", "Bowser"],
	"squad": "bros"
},
{
	"id": "Bowser",
	"squad": "koopas"
},
```
A cadence of one or two weeks is supported, with one week being the default. A two week cadence means that person will only be paired every second week.
```json
{
	"id": "Monty Mole",
    "cadence": "two-weeks"
}
```

## Development
[Golangci-lint](https://github.com/golangci/golangci-lint) is used for formatting/linting and must be installed separately.

```sh
# Run tests
go test ./...

# Check for linter/formatter issues
golangci-lint run

# Fix any issues which are auto-fixable
golangci-lint run --fix
```
