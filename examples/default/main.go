package main

import (
	"context"
	"fmt"

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
	// GET request
	posts, err := swiftreq.Get[[]Post](BASE_URL + "/posts").Do(context.Background())
	if err != nil {
		fmt.Println("Failed to GET", err)
	} else {
		fmt.Printf("\nGET Response: %+v\n", (*posts)[0])
	}

	post := Post{
		ID:    1,
		Title: "test title",
		Body:  "test body",
	}

	// POST request
	resp, err := swiftreq.Post[Post](BASE_URL+"/posts", post).Do(context.Background())
	if err != nil {
		fmt.Println("Failed to POST", err)
	} else {
		fmt.Printf("\nPOST Response: %+v\n", *resp)
	}

	// PUT request
	resp, err = swiftreq.Put[Post](BASE_URL+"/posts/1", post).Do(context.Background())
	if err != nil {
		fmt.Println("Failed to PUT", err)
	} else {
		fmt.Printf("\nPUT Response:  %+v\n", *resp)
	}

	// DELETE request
	resp, err = swiftreq.Delete[Post](BASE_URL + "/posts/1").Do(context.Background())
	if err != nil {
		fmt.Println("Failed to DELETE", err)
	} else {
		fmt.Printf("\nDELETE Response: %+v\n", *resp)
	}

}
