package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/vizicist/palette/kit"
)

func main() {
	flag.Parse()
	args := flag.Args()

	kit.InitLog("palette_hub")
	kit.InitKit()

	kit.LogInfo("Palette_Hub starting", "args", args)

	apiout, err := HubCommand(args)
	if err != nil {
		os.Stdout.WriteString("Error: " + err.Error() + "\n")
		kit.LogError(err)
		os.Exit(1)
	} else {
		os.Stdout.WriteString(kit.HumanReadableApiOutput(apiout))
	}
}

func usage() string {
	return `Usage:
	palette_hub streams
	palette_hub dumpraw [ {streamname} ]
	palette_hub dumpload [ {streamname} ]
	palette_hub dumpday {date} [ {streamname} ]
	  Date formats: 2025-12-11, 12-11, 12/11, today, yesterday
	palette_hub dumpdays [ {streamname} ]
	  Creates days/*.json files for each day from 2025-01-01 to yesterday
	`
}

func HubCommand(args []string) (map[string]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", usage())
	}

	// Connect to the remote NATS server
	err := kit.NatsConnectRemote()
	if err != nil {
		return map[string]string{"error": err.Error()}, nil
	}

	cmd := args[0]

	switch cmd {

	case "streams":
		streams, err := kit.NatsStreams()
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		s := ""
		for _, stream := range streams {
			s += fmt.Sprintf("%s\n", stream)
		}
		return map[string]string{"result": s}, nil

	case "dumpraw":
		streamName := "from_palette"
		if len(args) > 1 {
			streamName = args[1]
		}
		type DumpData struct {
			Subject string `json:"subject"`
			Tm      string `json:"time"`
			Data    string `json:"data"`
		}
		err := kit.NatsDump(streamName, func(tm time.Time, subj string, data string) {
			dd := DumpData{
				Subject: subj,
				Tm:      tm.Format(kit.PaletteTimeLayout),
				Data:    data,
			}
			jsonData, err := json.Marshal(dd)
			if err != nil {
				fmt.Println("Error marshalling JSON:", err)
				return
			}

			fmt.Println(string(jsonData))
		})
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		return map[string]string{"result": ""}, nil

	case "dumpload":
		streamName := "from_palette"
		if len(args) > 1 {
			streamName = args[1]
		}
		err = kit.NatsDump(streamName, func(tm time.Time, subj string, data string) {

			// We only look at .load messages
			if !strings.HasSuffix(subj, ".load") {
				return
			}

			var toplevel map[string]any
			err := json.Unmarshal([]byte(data), &toplevel)
			if err != nil {
				return
			}
			host := strings.TrimPrefix(subj, streamName+".")
			host = strings.TrimSuffix(host, ".load")

			// We used to include an attractmode flag in the published .load message,
			// but now we don't; we assume that attractmode loads won't even be published.
			// This code handles old logs that have the explicit attractmode value.
			a := toplevel["attractmode"]
			if a != nil {
				attractMode, ok := a.(bool)
				if !ok {
					kit.LogError(fmt.Errorf("bad attractmode value"))
					return
				}
				// If we're in attract mode, we ignore the load
				if attractMode {
					return
				}
			}

			f := toplevel["filename"]
			filename, ok := f.(string)
			if !ok {
				kit.LogError(fmt.Errorf("bad filename value"))
				return
			}
			if filename == "_Current" {
				return
			}

			c := toplevel["category"]
			category, ok := c.(string)
			if !ok {
				kit.LogError(fmt.Errorf("bad category value"))
				return
			}

			type DumpData struct {
				Event    string `json:"event"`
				Host     string `json:"host"`
				Category string `json:"category"`
				Tm       string `json:"time"`
				Filename string `json:"filename"`
			}

			dd := DumpData{
				Event:    "load",
				Host:     host,
				Tm:       tm.Format(kit.PaletteTimeLayout),
				Category: category,
				Filename: filename,
			}
			jsonData, err := json.Marshal(dd)
			if err != nil {
				fmt.Println("Error marshalling JSON:", err)
				return
			}

			fmt.Println(string(jsonData))

		})
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		return map[string]string{"result": ""}, nil

	case "dumpday":
		if len(args) < 2 {
			return nil, fmt.Errorf("dumpday requires a date argument\n%s", usage())
		}
		dateStr := args[1]
		streamName := "from_palette"
		if len(args) > 2 {
			streamName = args[2]
		}

		// Parse the date flexibly
		targetDate, err := parseFlexibleDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %s\n%s", err.Error(), usage())
		}

		// Set time range for the entire day (00:00:00 to 23:59:59.999999999)
		startTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
		endTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 999999999, time.UTC)

		type DumpData struct {
			Subject string `json:"subject"`
			Tm      string `json:"time"`
			Data    string `json:"data"`
		}

		err = kit.NatsDumpTimeRange(streamName, &startTime, &endTime, func(tm time.Time, subj string, data string) {
			dd := DumpData{
				Subject: subj,
				Tm:      tm.Format(kit.PaletteTimeLayout),
				Data:    data,
			}
			jsonData, err := json.Marshal(dd)
			if err != nil {
				fmt.Println("Error marshalling JSON:", err)
				return
			}

			fmt.Println(string(jsonData))
		})
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		return map[string]string{"result": ""}, nil

	case "dumpdays":
		streamName := "from_palette"
		if len(args) > 1 {
			streamName = args[1]
		}

		err := dumpDays(streamName)
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		return map[string]string{"result": "Daily dumps completed"}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s\n%s", cmd, usage())
	}
}

// dumpDays creates daily dump files in the days/ directory
func dumpDays(streamName string) error {
	// Create days directory if it doesn't exist
	daysDir := "days"
	if err := os.MkdirAll(daysDir, 0755); err != nil {
		return fmt.Errorf("failed to create days directory: %v", err)
	}

	// Define start and end dates in local timezone
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// Fallback to local time if timezone loading fails
		loc = time.Local
	}

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, loc)
	yesterday := time.Now().In(loc).AddDate(0, 0, -1)
	endDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, loc)

	// Iterate through each day
	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, 1) {
		dateStr := currentDate.Format("2006-01-02")
		filename := fmt.Sprintf("%s/%s.json", daysDir, dateStr)

		// Check if file already exists
		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("Skipping %s (already exists)\n", dateStr)
			continue
		}

		fmt.Printf("Dumping %s...\n", dateStr)

		// Create the file
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filename, err)
		}

		// Set time range for the entire day in local timezone (Pacific Time)
		dayStart := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, loc)
		dayEnd := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 999999999, loc)

		type DumpData struct {
			Subject string `json:"subject"`
			Tm      string `json:"time"`
			Data    string `json:"data"`
		}

		messageCount := 0

		// Dump messages for this day
		err = kit.NatsDumpTimeRange(streamName, &dayStart, &dayEnd, func(tm time.Time, subj string, data string) {
			dd := DumpData{
				Subject: subj,
				Tm:      tm.Format(kit.PaletteTimeLayout),
				Data:    data,
			}
			jsonData, err := json.Marshal(dd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
				return
			}

			file.WriteString(string(jsonData) + "\n")
			messageCount++
		})

		file.Close()

		if err != nil {
			return fmt.Errorf("error dumping %s: %v", dateStr, err)
		}

		fmt.Printf("  -> %d messages written to %s\n", messageCount, filename)
	}

	return nil
}

// parseFlexibleDate parses various date formats and returns a time.Time
func parseFlexibleDate(dateStr string) (time.Time, error) {
	now := time.Now().UTC()

	// Handle special keywords
	switch strings.ToLower(dateStr) {
	case "today":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}

	// Try various date formats
	formats := []string{
		"2006-01-02",     // 2025-12-11
		"2006/01/02",     // 2025/12/11
		"01-02",          // 12-11 (assumes current year)
		"01/02",          // 12/11 (assumes current year)
		"01-02-2006",     // 12-11-2025
		"01/02/2006",     // 12/11/2025
		"2006-01-02T15:04:05Z07:00", // RFC3339
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			// If format doesn't include year, use current year
			if format == "01-02" || format == "01/02" {
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", dateStr)
}
