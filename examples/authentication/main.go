package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/liviudnicoara/swiftreq"
)

func main() {
	startServerWithAuthentication()
	time.Sleep(1 * time.Second)

	re := swiftreq.Default().WithAuthorization("Token", func() (token string, lifeSpan time.Duration, err error) {
		resp, err := swiftreq.Get[string]("http://localhost:3000/auth").
			WithRequestExecutor(swiftreq.NewRequestExecutor(*http.DefaultClient)).
			WithHeaders(map[string]string{"Credentials": "user:pass"}).
			Do(context.Background())

		vals := strings.Split(*resp, " ")

		lifeSpanUnix, _ := strconv.ParseInt(vals[1], 10, 64)
		ls := time.Unix(lifeSpanUnix, 0)

		return vals[0], time.Until(ls), err
	})

	time.Sleep(1 * time.Second)

	resp, err := swiftreq.Get[string]("http://localhost:3000/page").
		WithRequestExecutor(re).
		Do(context.Background())

	if err != nil {
		fmt.Println("ERROR", err)
	}

	fmt.Println("RESPONSE", *resp)

}

func startServerWithAuthentication() {
	mux := http.NewServeMux()

	var expiredAt time.Time
	mux.Handle("/auth", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credentials := r.Header.Get("Credentials")
		fmt.Println("Auth requested with credentials", credentials)

		token := base64.StdEncoding.EncodeToString([]byte(credentials))
		expiredAt = time.Now().Add(100 * time.Second)
		token = fmt.Sprintf("%s %d", token, expiredAt.Unix())

		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte(token))
	}))

	mux.Handle("/page", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Received request with Authorization", r.Header.Get("Authorization"))
		fmt.Println("Expire time ", expiredAt)
		if time.Now().After(expiredAt) {
			fmt.Println("Token expired")
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Add("Content-Type", "text/plain")
			w.Write([]byte("Unauthorized"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("Success"))
	}))

	go func() {
		if err := http.ListenAndServe(":3000", mux); err != nil {
			log.Fatal(err)
		}
	}()
}
