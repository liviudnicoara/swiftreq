package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/liviudnicoara/swiftreq"
)

func main() {
	mux := http.NewServeMux()

	var retry int
	var retryStart time.Time
	mux.Handle("/retry", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retry == 0 {
			retryStart = time.Now()
		}

		fmt.Printf("URL: /retry. Retry %d after %s\n", retry, time.Since(retryStart))

		if retry == 3 {
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "text/plain")
			w.Write([]byte("OK"))
			retry = 0
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("Error"))
		retry++
	}))

	mux.Handle("/error", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("Error"))
		retry++
	}))

	retryAfter := -1
	mux.Handle("/retry-after", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retryAfter == 3 {
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "text/plain")
			w.Write([]byte("OK"))
			retry = 0
			return
		}

		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Retry-After", "5")
		w.Write([]byte("too many request"))
		retry++
	}))

	go func() {
		if err := http.ListenAndServe(":3000", mux); err != nil {
			log.Fatal(err)
		}
	}()

	resp, err := swiftreq.NewGetRequest[string]("http://localhost:3000/retry").
		WithRequestExecutor(swiftreq.Default().
			WithExponentialRetry(5)).
		Do(context.Background())

	if err != nil {
		fmt.Println("ERROR", err)
	}

	fmt.Println("RESPONSE", *resp)

}
