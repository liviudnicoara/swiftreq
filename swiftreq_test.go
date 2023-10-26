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
		case "/error":
			mockErrorEndpoint(w, r)
		case "/timeout":
			mockTimeoutEndpoint(w, r)
		case "/post":
			mockPostEndpoint(w, r)
		case "/post/error":
			mockErrorEndpoint(w, r)
		case "/put":
			mockPostEndpoint(w, r)
		case "/put/error":
			mockErrorEndpoint(w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))

	fmt.Println("run tests")
	m.Run()
}

func mockErrorEndpoint(w http.ResponseWriter, r *http.Request) {
	sc := http.StatusBadRequest
	m := make(map[string]interface{})

	m["error"] = "custom endpoint error"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	json.NewEncoder(w).Encode(m)
}

func mockGetEndpoint(w http.ResponseWriter, r *http.Request) {
	idString := r.URL.Query().Get("id")

	sc := http.StatusOK
	m := make(map[string]interface{})

	id, _ := strconv.Atoi(idString)
	m["id"] = id
	m["name"] = "mock"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	json.NewEncoder(w).Encode(m)
}

func mockTimeoutEndpoint(w http.ResponseWriter, r *http.Request) {
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

	m["id"] = req.ID
	m["name"] = "mock"

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
		resp, err := req.Get(context.Background(), server.URL+"/error")

		// assert
		assert.Contains(t, err.Error(), "custom endpoint error")
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
		body := TestRequest{
			ID:   0,
			Type: "user",
		}

		// act
		resp, err := swiftreq.NewDefaultRequest[TestResponse]().Post(context.Background(), server.URL+"/post/error", &body)

		// assert
		assert.Contains(t, err.Error(), "custom endpoint error")
		assert.Nil(t, resp)
	})
}

func Test_Put(t *testing.T) {
	t.Run("Sucess", func(t *testing.T) {
		// arrange
		req := swiftreq.NewDefaultRequest[TestResponse]()
		body := TestRequest{
			ID:   1,
			Type: "user",
		}

		// act
		resp, err := req.Put(context.Background(), server.URL+"/put", &body)

		// assert
		assert.Equal(t, 1, resp.ID)
		assert.Equal(t, "mock", resp.Name)
		assert.Nil(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// arrange
		body := TestRequest{
			ID:   0,
			Type: "user",
		}

		// act
		resp, err := swiftreq.NewDefaultRequest[TestResponse]().Put(context.Background(), server.URL+"/put/error", &body)

		// assert
		assert.Contains(t, err.Error(), "custom endpoint error")
		assert.Nil(t, resp)
	})
}
