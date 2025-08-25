package tools

import (
	"context"
	"net/http"
	"time"

	ics "github.com/arran4/golang-ical"
)

// LoadCalendar loads calendar events from a URL
func LoadCalendar(ctx context.Context, url string) ([]*ics.VEvent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cal, err := ics.ParseCalendar(resp.Body)
	if err != nil {
		return nil, err
	}

	var events []*ics.VEvent
	for _, component := range cal.Components {
		if event, ok := component.(*ics.VEvent); ok {
			events = append(events, event)
		}
	}

	return events, nil
}