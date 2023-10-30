package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/liviudnicoara/swiftreq"
)

const BASE_URL = "https://jsonplaceholder.typicode.com"

type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

func main() {
	// Create custom rest executor
	re := swiftreq.NewDefaultRequestExecutor(). // default executor with 30s timeout
							AddLogging(*slog.Default()).                                // add logger
							AddPerformanceMonitor(10*time.Millisecond, *slog.Default()) // add performance monitor

	// GET request
	post, err := swiftreq.NewRequest[Post](re).
		WithURL(BASE_URL + "/posts/1").
		WithMethod("GET").
		WithQueryParameters(map[string]string{"page": "1"}).
		WithHeaders(map[string]string{"Content-Type": "application/json"}).
		Do(context.Background())

	if err != nil {
		fmt.Println("Failed to GET", err)
	} else {
		fmt.Printf("\nGET Response: %+v\n", post)
	}

}
