package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/alexflint/go-arg"
	ics "github.com/arran4/golang-ical"
)

const DateTimeFormat = "20060102T150405"

type Config struct {
	Original string
	Path     string
	Events   map[string]Event
	Global   []GlobalRule
}

type Event struct {
	Summary string
	Changes []Change
}

type Change struct {
	Date     time.Time
	Canceled bool
	Location string
}

type GlobalRule struct {
	Match   string
	Replace string
}

func main() {
	var args struct {
		Config string `help:"location of config file" default:"transformer.toml"`
		Print  bool   `help:"set to print events of original url"`
		Server bool   `help:"set to serve the transformed iCal/Webcal"`
	}
	arg.MustParse(&args)

	if args.Print == args.Server {
		log.Fatal("Program should run in either server or print mode.")
	}

	fmt.Println("iCal Transformer")

	config := getConfig(args.Config)

	if args.Print {
		cal := getOriginal(config)
		printAllEvents(cal.Events())
	} else if args.Server {
		serveServer(config)
	}

}

func getConfig(path string) Config {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		log.Fatalf("Error while reading config file: %s\n", err)
	}

	return config
}

func serveServer(config Config) {
	addr := "0.0.0.0:8080"

	log.Printf("Listing on %s with path '%s'...\n", addr, config.Path)

	http.HandleFunc(config.Path, func(writer http.ResponseWriter, request *http.Request) {
		cal := getOriginal(config)
		err := processTransforms(cal, config)
		if err != nil {
			log.Printf("Error while processing %s: %s\n", config.Path, err)
		}

		_, err = fmt.Fprint(writer, cal.Serialize())
		if err != nil {
			log.Fatalf("error while printing to response writer: %s\n", err)
		}
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}

func getOriginal(config Config) *ics.Calendar {
	res, err := http.Get(config.Original)
	if err != nil {
		log.Fatalf("Error making http request: %s\n", err)
	}

	cal, err := ics.ParseCalendar(res.Body)
	if err != nil {
		log.Fatalf("Error while parsing ical: %s\n", err)
	}

	err = res.Body.Close()
	if err != nil {
		log.Fatalf("Error closing response body: %s\n", err)
	}

	return cal
}

func processTransforms(cal *ics.Calendar, config Config) error {
	if err := processGlobal(cal, config); err != nil {
		return err
	}
	if err := processEvents(cal, config); err != nil {
		return err
	}
	return nil
}

func processEvents(cal *ics.Calendar, config Config) error {
	for uid, configEvent := range config.Events {
		event := getEventByUID(cal.Components, uid)
		if event == nil {
			continue
		}

		if configEvent.Summary != "" {
			event.SetSummary(configEvent.Summary)
		}

		for i, change := range configEvent.Changes {
			if change.Canceled {
				startTime, err := event.GetStartAt()
				if err != nil {
					return err
				}
				event.AddExdate(getAlignedTime(startTime, change.Date).Format(DateTimeFormat),
					getTimeZoneProperty(startTime))
				continue
			}

			if change.Location != "" {
				startTime, err := event.GetStartAt()
				if err != nil {
					return err
				}
				event.AddExdate(getAlignedTime(startTime, change.Date).Format(DateTimeFormat),
					getTimeZoneProperty(startTime))

				summary := ""
				if configEvent.Summary != "" {
					summary = configEvent.Summary
				} else if summaryBody := event.GetProperty(ics.ComponentPropertySummary); summaryBody != nil {
					summary = summaryBody.Value
				}

				description := ""
				oldDescriptionProp := event.GetProperty(ics.ComponentPropertyDescription)
				if oldDescriptionProp != nil {
					description = oldDescriptionProp.Value
				}

				newEvent := ics.NewEvent(event.Id() + "-ical-transformer-" + fmt.Sprint(i))
				newEvent.SetSummary(summary)
				newEvent.SetLocation(change.Location)
				newEvent.SetDescription(description)

				dtStampTime, err := event.GetDtStampTime()
				if err != nil {
					return err
				}
				newEvent.SetDtStampTime(dtStampTime)

				newStartTime := getAlignedTime(startTime, change.Date)
				newEvent.SetStartAt(newStartTime)

				endTime, err := event.GetEndAt()
				if err != nil {
					return err
				}
				newEvent.SetEndAt(getAlignedTime(endTime, change.Date))

				cal.Components = append(cal.Components, newEvent)
			}
		}
	}
	return nil
}

func processGlobal(cal *ics.Calendar, config Config) error {
	for _, rule := range config.Global {
		r, _ := regexp.Compile(rule.Match)

		for _, event := range cal.Events() {
			summaryProp := event.GetProperty(ics.ComponentPropertySummary)
			if summaryProp != nil {
				event.SetSummary(r.ReplaceAllString(summaryProp.Value, rule.Replace))
			}

		}
	}
	return nil
}

func getTimeZoneProperty(date time.Time) ics.PropertyParameter {
	return ics.WithTZID(date.Location().String())
}

func getAlignedTime(eventTime time.Time, date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), eventTime.Hour(), eventTime.Minute(), 0, 0, eventTime.Location())
}

func printAllEvents(events []*ics.VEvent) {
	for _, event := range events {
		uniqueId := event.GetProperty(ics.ComponentPropertyUniqueId)
		if uniqueId == nil {
			continue
		}

		fmt.Printf("Event (%s)\n", uniqueId.Value)
		for _, property := range event.Properties {
			fmt.Printf(" - %s: %s\n", property.IANAToken, property.Value)
		}
	}
}

func getEventByUID(components []ics.Component, uid string) *ics.VEvent {
	for _, component := range components {
		switch event := component.(type) {
		case *ics.VEvent:
			if event.GetProperty(ics.ComponentPropertyUniqueId).Value == uid {
				return event
			}
		}
	}
	return nil
}
