package swiftreq_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/liviudnicoara/swiftreq"
	"github.com/stretchr/testify/assert"
)

var (
	server *httptest.Server
)

type TestRequest struct {
	ID   int
	Type string
}

type TestResponse struct {
	ID   int
	Name string
}

func TestMain(m *testing.M) {
	fmt.Println("mocking server")
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			mockGetEndpoint(w, r)
		case "/timeout":
			mockGetTimeoutEndpoint(w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))

	fmt.Println("run tests")
	m.Run()
}

func mockGetEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	sc := http.StatusOK
	m := make(map[string]interface{})

	if id == "" {
		sc = http.StatusBadRequest
		m["error"] = "missing id"
	} else {
		id, _ := strconv.Atoi(id)
		m["id"] = id
		m["name"] = "mock"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	json.NewEncoder(w).Encode(m)
}

func mockGetTimeoutEndpoint(w http.ResponseWriter, r *http.Request) {
	sc := http.StatusOK
	m := make(map[string]interface{})

	m["id"] = 1
	m["name"] = "mock"

	time.Sleep(1 * time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	json.NewEncoder(w).Encode(m)
}

func Test_Get(t *testing.T) {
	t.Run("Sucess", func(t *testing.T) {
		// arrange
		re := swiftreq.NewRequestExecutor(http.Client{})
		req := swiftreq.NewRequest[swiftreq.EmptyRequest, TestResponse](re)
		query := make(url.Values)
		query.Add("id", "1")

		// act
		resp, err := req.Get(context.Background(), server.URL, nil, query, nil)

		// assert
		assert.Equal(t, 1, resp.ID)
		assert.Equal(t, "mock", resp.Name)
		assert.Nil(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// arrange
		re := swiftreq.NewRequestExecutor(http.Client{})
		req := swiftreq.NewRequest[swiftreq.EmptyRequest, TestResponse](re)
		query := make(url.Values)

		// act
		resp, err := req.Get(context.Background(), server.URL, nil, query, nil)

		// assert
		assert.Contains(t, err.Error(), "missing id")
		assert.Nil(t, resp)
	})

	t.Run("ExecutorTimeout", func(t *testing.T) {
		// arrange
		re := swiftreq.NewRequestExecutor(http.Client{Timeout: 100 * time.Millisecond})
		req := swiftreq.NewRequest[swiftreq.EmptyRequest, TestResponse](re)
		query := make(url.Values)

		// act
		resp, err := req.Get(context.Background(), server.URL+"/timeout", nil, query, nil)

		// assert
		assert.Contains(t, err.Error(), "deadline exceeded")
		assert.Nil(t, resp)
	})
}
