package swiftreq_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
		case "/post":
			mockPostEndpoint(w, r)
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

	time.Sleep(200 * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	json.NewEncoder(w).Encode(m)
}

func mockPostEndpoint(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var req TestRequest
	_ = json.Unmarshal(body, &req)

	sc := http.StatusOK
	m := make(map[string]interface{})

	if req.ID == 0 {
		sc = http.StatusBadRequest
		m["error"] = "missing id"
	} else {
		m["id"] = req.ID
		m["name"] = "mock"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	json.NewEncoder(w).Encode(m)
}

func Test_Get(t *testing.T) {
	t.Run("Sucess", func(t *testing.T) {
		// arrange
		query := map[string]string{
			"id": "1",
		}
		req := swiftreq.NewDefaultRequest[TestResponse]().WithQueryParameters(query)

		// act
		resp, err := req.Get(context.Background(), server.URL)

		// assert
		assert.Equal(t, 1, resp.ID)
		assert.Equal(t, "mock", resp.Name)
		assert.Nil(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// arrange
		req := swiftreq.NewDefaultRequest[TestResponse]()

		// act
		resp, err := req.Get(context.Background(), server.URL)

		// assert
		assert.Contains(t, err.Error(), "missing id")
		assert.Nil(t, resp)
	})

	t.Run("ExecutorTimeout", func(t *testing.T) {
		// arrange
		re := swiftreq.NewRequestExecutor(http.Client{Timeout: 100 * time.Millisecond})
		req := swiftreq.NewRequest[TestResponse](re)

		// act
		resp, err := req.Get(context.Background(), server.URL+"/timeout")

		// assert
		assert.Contains(t, err.Error(), "deadline exceeded")
		assert.Nil(t, resp)
	})
}

func Test_Post(t *testing.T) {
	t.Run("Sucess", func(t *testing.T) {
		// arrange
		req := swiftreq.NewDefaultRequest[TestResponse]()
		body := TestRequest{
			ID:   1,
			Type: "user",
		}

		// act
		resp, err := req.Post(context.Background(), server.URL+"/post", &body)

		// assert
		assert.Equal(t, 1, resp.ID)
		assert.Equal(t, "mock", resp.Name)
		assert.Nil(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// arrange
		req := swiftreq.NewDefaultRequest[TestResponse]()
		body := TestRequest{
			ID:   0,
			Type: "user",
		}

		// act
		resp, err := req.Post(context.Background(), server.URL+"/post", &body)

		// assert
		assert.Contains(t, err.Error(), "missing id")
		assert.Nil(t, resp)
	})
}
