package yandex

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/X1ag/TravelScheduler/internal/domain"
)

// rostov code - 9612913
// taganrog code - 9613483
// api code - 6f7478e5-151e-436d-b8ba-ace9a4c05375

// https://api.rasp.yandex-net.ru/v3.0/search/?apikey=6f7478e5-151e-436d-b8ba-ace9a4c05375&format=json&transport_types=suburban&from=s9613483&to=s9612913&lang=ru_RU&page=1&date=2026-01-23

type yandexResponse struct {
	Segments []struct {
		DepartureTime time.Time `json:"departure"`
		ArrivalTime time.Time `json:"arrival"`
		Duration float64 `json:"duration"`
		Thread struct {
			Number string `json:"number"`
			Title string `json:"title"`
		} `json:"thread"`
	} `json:"segments"`
}

type failderResponse struct {
 // TODO: implement failed
}

type Client struct {
	apiKey string
	client *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey, 
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetNextTrains(ctx context.Context, from, to string, date time.Time) ([]*domain.Schedule, error){
	if from[0] != 's' { from = fmt.Sprintf("s%s", from) }
	if to[0] != 's' { to = fmt.Sprintf("s%s", to) }

	url := fmt.Sprintf("https://api.rasp.yandex-net.ru/v3.0/search/?apikey=%s&format=json&transport_types=suburban&from=%s&to=%s&lang=ru_RU&page=1&date=%s", c.apiKey, from, to, date.Format("2006-01-02"))

	log.Println(url)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// TODO: return failed "text" 
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var data yandexResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var options []*domain.Schedule 
	for _, s := range data.Segments {
		options = append(options, &domain.Schedule{
			TrainID: s.Thread.Number,
			Title: s.Thread.Title,
			DepartureTime: s.DepartureTime,
			ArrivalTime: s.ArrivalTime,
			Duration: s.Duration,
		})
	}

	return options, nil
} 