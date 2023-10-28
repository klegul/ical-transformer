package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/alexflint/go-arg"
	ics "github.com/arran4/golang-ical"
	"log"
	"net/http"
)

type Config struct {
	Original string
	Path     string
	Events   map[string]Event
}

type Event struct {
	Summary     string
	Exdates_add []string
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
		processTransforms(cal, config)

		_, err := fmt.Fprint(writer, cal.Serialize())
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

	return cal
}

func processTransforms(cal *ics.Calendar, config Config) {
	for uid, configEvent := range config.Events {
		event := getEventByUID(cal.Components, uid)
		if event == nil {
			continue
		}

		if configEvent.Summary != "" {
			event.SetSummary(configEvent.Summary)
		}

		if len(configEvent.Exdates_add) > 0 {
			for _, exdate := range configEvent.Exdates_add {
				event.AddExdate(exdate)
			}
		}
	}
}

func printAllEvents(events []*ics.VEvent) {
	for _, event := range events {
		fmt.Printf("Event (%s)\n", event.GetProperty(ics.ComponentPropertyUniqueId).Value)
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
