package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	json "github.com/goccy/go-json"
	"github.com/nats-io/nats.go"
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
	palette_hub listen [ {streamname} ]
	  Subscribe and print events in real-time (Ctrl+C to stop)
	palette_hub request_log {hostname} [ start={time} ] [ end={time} ] [ limit={n} ]
	  Request log entries from a palette via NATS
	  Time format: RFC3339 (e.g., 2026-01-30T00:00:00Z)
	  Example: palette_hub request_log spacepalette34 limit=100
	palette_hub dumpraw [ {streamname} ]
	palette_hub dumpload [ {streamname} ]
	palette_hub dumpday {date} [ {streamname} ]
	  Date formats: 2025-12-11, 12-11, 12/11, today, yesterday
	palette_hub dumpdays [ {streamname} ]
	  Creates days/*.json files for each day from 2025-01-01 to yesterday
	palette_hub import_log {hostname}
	  Reads engine.log from stdin and merges events into days/*.json files
	  Deduplicates against existing events in the days files
	  Example: cat engine.log | ssh hub_machine "cd palette_hub && ./palette_hub import_log spacepalette37"
	`
}

func HubCommand(args []string) (map[string]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", usage())
	}

	cmd := args[0]

	// Handle commands that don't need NATS connection
	if cmd == "import_log" {
		if len(args) < 2 {
			return nil, fmt.Errorf("import_log requires a hostname argument\n%s", usage())
		}
		hostname := args[1]
		result, err := importEngineLog(hostname)
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		return map[string]string{"result": result}, nil
	}

	// Connect to the remote NATS server for other commands
	err := kit.NatsConnectRemote()
	if err != nil {
		return map[string]string{"error": err.Error()}, nil
	}

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

	case "listen":
		// Subscribe to events in real-time
		subject := ">"
		if len(args) > 1 {
			subject = args[1] + ".>"
		}

		type EventData struct {
			Subject string `json:"subject"`
			Tm      string `json:"time"`
			Data    string `json:"data"`
		}

		fmt.Printf("Listening to %s (Ctrl+C to stop)...\n", subject)

		err := kit.NatsSubscribe(subject, func(msg *nats.Msg) {
			ed := EventData{
				Subject: msg.Subject,
				Tm:      time.Now().Format(kit.PaletteTimeLayout),
				Data:    string(msg.Data),
			}
			jsonData, err := json.Marshal(ed)
			if err != nil {
				fmt.Println("Error marshalling JSON:", err)
				return
			}
			fmt.Println(string(jsonData))
		})
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}

		// Wait for Ctrl+C
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nStopped listening.")
		return map[string]string{"result": ""}, nil

	case "request_log":
		if len(args) < 2 {
			return nil, fmt.Errorf("request_log requires a hostname argument\n%s", usage())
		}
		hostname := args[1]

		// Parse optional key=value arguments
		params := make(map[string]string)
		for _, arg := range args[2:] {
			if parts := strings.SplitN(arg, "=", 2); len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}

		// Fetch log entries in batches
		timeout := 5 * time.Second
		batchSize := 500
		offset := 0
		totalEntries := 0

		for {
			// Build the API request for this batch
			apiRequest := map[string]string{
				"api":    "global.log",
				"limit":  strconv.Itoa(batchSize),
				"offset": strconv.Itoa(offset),
			}
			if v, ok := params["start"]; ok {
				apiRequest["start"] = v
			}
			if v, ok := params["end"]; ok {
				apiRequest["end"] = v
			}

			requestJSON, err := json.Marshal(apiRequest)
			if err != nil {
				return map[string]string{"error": err.Error()}, nil
			}

			response, err := kit.EngineNatsApi(hostname, string(requestJSON), timeout)
			if err != nil {
				return map[string]string{"error": fmt.Sprintf("NATS request failed: %v", err)}, nil
			}

			// Parse the response to check for errors
			var responseData map[string]any
			if err := json.Unmarshal([]byte(response), &responseData); err != nil {
				// Not JSON, just output as-is
				fmt.Println(response)
				return map[string]string{"result": ""}, nil
			}

			// Check if response has an error
			if errMsg, ok := responseData["error"].(string); ok {
				return map[string]string{"error": errMsg}, nil
			}

			// Check if response has a result field with the log entries
			result, ok := responseData["result"].(string)
			if !ok {
				// Output raw response and stop
				fmt.Println(response)
				break
			}

			// Parse the log entries
			var entries []map[string]any
			if err := json.Unmarshal([]byte(result), &entries); err != nil {
				// Not a JSON array, output as-is and stop
				fmt.Println(result)
				break
			}

			// Output each entry as a separate JSON line, converting time to UTC
			for _, entry := range entries {
				// Convert time field to UTC if present
				if timeStr, ok := entry["time"].(string); ok {
					if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
						entry["time"] = t.UTC().Format(time.RFC3339)
					}
				}
				entryJSON, _ := json.Marshal(entry)
				fmt.Println(string(entryJSON))
			}

			totalEntries += len(entries)

			// If we got fewer entries than the batch size, we're done
			if len(entries) < batchSize {
				break
			}

			offset += batchSize
		}

		fmt.Fprintf(os.Stderr, "Total entries: %d\n", totalEntries)
		return map[string]string{"result": ""}, nil

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
		return map[string]string{"result": "Daily dumps completed\n"}, nil

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

	// Define start and end dates in UTC
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	endDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)

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

		// Set time range for the entire day in UTC
		dayStart := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 999999999, time.UTC)

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
		"2006-01-02",                // 2025-12-11
		"2006/01/02",                // 2025/12/11
		"01-02",                     // 12-11 (assumes current year)
		"01/02",                     // 12/11 (assumes current year)
		"01-02-2006",                // 12-11-2025
		"01/02/2006",                // 12/11/2025
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

// importEngineLog reads an engine.log from stdin and merges events into days files
func importEngineLog(hostname string) (string, error) {
	// Create days directory if it doesn't exist
	daysDir := "days"
	if err := os.MkdirAll(daysDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create days directory: %v", err)
	}

	// Read all lines from stdin
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for potentially long log lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var startTime time.Time
	var events []DayEvent

	// Track attract mode state - loads during attract mode should be skipped
	// (matching the behavior of NatsPublishFromEngine which only publishes when !isOn)
	attractModeOn := false

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var logEntry map[string]any
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue // Skip non-JSON lines
		}

		msg, ok := logEntry["msg"].(string)
		if !ok {
			continue
		}

		uptimeStr, ok := logEntry["uptime"].(string)
		if !ok {
			continue
		}
		uptime, err := strconv.ParseFloat(uptimeStr, 64)
		if err != nil {
			continue
		}

		// Look for InitLog to get start time
		if msg == "InitLog ==============================" {
			timeStr, ok := logEntry["time"].(string)
			if ok {
				t, err := time.Parse(kit.PaletteTimeLayout, timeStr)
				if err == nil {
					// Subtract uptime to get the actual start time
					startTime = t.Add(-time.Duration(uptime * float64(time.Second)))
					// Reset attract mode state on new session
					attractModeOn = false
				}
			}
			continue
		}

		// Skip if we haven't found a start time yet
		if startTime.IsZero() {
			continue
		}

		// Calculate absolute time for this event
		eventTime := startTime.Add(time.Duration(uptime * float64(time.Second)))

		// Extract attract mode events
		if msg == "setAttractMode" {
			onoff, ok := logEntry["onoff"].(bool)
			if !ok {
				continue
			}
			// Update our tracking of attract mode state
			attractModeOn = onoff
			data := map[string]any{"onoff": onoff}
			dataBytes, _ := json.Marshal(data)
			events = append(events, DayEvent{
				Subject: fmt.Sprintf("from_palette.%s.attract", hostname),
				Time:    eventTime,
				Data:    string(dataBytes),
			})
		}

		// Extract load events - but only when NOT in attract mode
		// This matches the NATS publishing logic in kit/quad.go
		if msg == "Quad.Load" {
			// Skip loads during attract mode (these wouldn't have been published via NATS)
			if attractModeOn {
				continue
			}
			category, ok1 := logEntry["category"].(string)
			filename, ok2 := logEntry["filename"].(string)
			if !ok1 || !ok2 {
				continue
			}
			// Skip _Current loads
			if filename == "_Current" {
				continue
			}
			data := map[string]any{"category": category, "filename": filename}
			dataBytes, _ := json.Marshal(data)
			events = append(events, DayEvent{
				Subject: fmt.Sprintf("from_palette.%s.load", hostname),
				Time:    eventTime,
				Data:    string(dataBytes),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stdin: %v", err)
	}

	if len(events) == 0 {
		return "No events found in engine.log\n", nil
	}

	// Group events by day
	eventsByDay := make(map[string][]DayEvent)
	for _, event := range events {
		dayStr := event.Time.UTC().Format("2006-01-02")
		eventsByDay[dayStr] = append(eventsByDay[dayStr], event)
	}

	// Process each day
	totalNew := 0
	totalSkipped := 0
	daysModified := 0

	for dayStr, dayEvents := range eventsByDay {
		filename := fmt.Sprintf("%s/%s.json", daysDir, dayStr)

		// Load existing events from the day file (if it exists)
		existingEvents := make(map[string]bool)
		if fileData, err := os.ReadFile(filename); err == nil {
			lines := strings.Split(string(fileData), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				// Create a key from the event for deduplication
				existingEvents[line] = true
			}
		}

		// Filter out duplicates and prepare new events
		var newEvents []DayEvent
		for _, event := range dayEvents {
			eventLine := formatDayEvent(event)
			if !existingEvents[eventLine] {
				newEvents = append(newEvents, event)
			} else {
				totalSkipped++
			}
		}

		if len(newEvents) == 0 {
			continue
		}

		// Read existing file content (if any)
		var allEvents []DayEvent
		if fileData, err := os.ReadFile(filename); err == nil {
			lines := strings.Split(string(fileData), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				event, err := parseDayEvent(line)
				if err == nil {
					allEvents = append(allEvents, event)
				}
			}
		}

		// Add new events
		allEvents = append(allEvents, newEvents...)

		// Sort by time
		sort.Slice(allEvents, func(i, j int) bool {
			return allEvents[i].Time.Before(allEvents[j].Time)
		})

		// Write back to file
		file, err := os.Create(filename)
		if err != nil {
			return "", fmt.Errorf("failed to create file %s: %v", filename, err)
		}

		for _, event := range allEvents {
			file.WriteString(formatDayEvent(event) + "\n")
		}
		file.Close()

		totalNew += len(newEvents)
		daysModified++
		fmt.Printf("  %s: added %d events (total now %d)\n", dayStr, len(newEvents), len(allEvents))
	}

	return fmt.Sprintf("Imported %d new events, skipped %d duplicates, modified %d day files\n",
		totalNew, totalSkipped, daysModified), nil
}

// DayEvent represents an event to be stored in a day file
type DayEvent struct {
	Subject string
	Time    time.Time
	Data    string
}

// formatDayEvent formats an event as a JSON line for the day file
func formatDayEvent(event DayEvent) string {
	type DumpData struct {
		Subject string `json:"subject"`
		Tm      string `json:"time"`
		Data    string `json:"data"`
	}
	dd := DumpData{
		Subject: event.Subject,
		Tm:      event.Time.Format(kit.PaletteTimeLayout),
		Data:    event.Data,
	}
	jsonData, _ := json.Marshal(dd)
	return string(jsonData)
}

// parseDayEvent parses a JSON line from a day file
func parseDayEvent(line string) (DayEvent, error) {
	var dd struct {
		Subject string `json:"subject"`
		Tm      string `json:"time"`
		Data    string `json:"data"`
	}
	if err := json.Unmarshal([]byte(line), &dd); err != nil {
		return DayEvent{}, err
	}
	t, err := time.Parse(kit.PaletteTimeLayout, dd.Tm)
	if err != nil {
		return DayEvent{}, err
	}
	return DayEvent{
		Subject: dd.Subject,
		Time:    t,
		Data:    dd.Data,
	}, nil
}
