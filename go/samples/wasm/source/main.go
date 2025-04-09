package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/iot-operations-sdks/go/wasm/source"
)

func main() {}

func init() {
	source.Init(func() error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		runners := map[string]context.CancelFunc{}

		for event := range source.Poll(ctx, time.Millisecond) {
			switch e := event.(type) {
			case source.DatasetCreated:
				ctx, cancel := context.WithCancel(ctx)
				runners[e.Name] = cancel
				go runner(ctx, e.Dataset)

			case source.DatasetUpdated:
				if cancel, ok := runners[e.Name]; ok {
					cancel()
				}

				ctx, cancel := context.WithCancel(ctx)
				runners[e.Name] = cancel
				go runner(ctx, e.Dataset)

			case source.DatasetRemoved:
				if cancel, ok := runners[e.Name]; ok {
					cancel()
				}

				delete(runners, e.Name)
			}
		}

		return nil
	})
}

func runner(ctx context.Context, data *source.Dataset) {
	freq := time.Duration(data.PollFrequencyMs) * time.Millisecond
	for {
		select {
		case <-time.After(freq):
			if err := pushData(ctx, data); err != nil {
				fmt.Println(err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func pushData(ctx context.Context, data *source.Dataset) error {
	var cfg struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	}

	if err := json.Unmarshal(
		[]byte(data.AdditionalConfiguration),
		&cfg,
	); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	source.Send("dataset", data.Name, body)
	return nil
}
