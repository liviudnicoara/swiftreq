package main

import (
	"context"
	"fmt"
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
							AddCaching(100 * time.Millisecond)

	// GET request
	req := swiftreq.NewGetRequest[Post](BASE_URL + "/posts/1").
		WithRequestExecutor(re).
		WithQueryParameters(map[string]string{"page": "1"})

	start := time.Now()
	_, err := req.Do(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		fmt.Println("Failed to GET", err)
	} else {
		fmt.Println("Cache empty. Request took: ", elapsed)
	}

	start = time.Now()
	_, err = req.Do(context.Background())
	elapsed = time.Since(start)

	if err != nil {
		fmt.Println("Failed to GET", err)
	} else {
		fmt.Println("From cache. Request took: ", elapsed)
	}

}
