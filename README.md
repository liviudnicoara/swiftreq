![](https://raw.githubusercontent.com/liviudnicoara/liviudnicoara/main/assests/swiftreq.png)

# SwiftReq

SwiftReq is a Golang library designed to simplify HTTP requests in your services. It offers a set of features to enhance your interaction with external services.

## Features

- **Generics:** Utilize generics for seamless HTTP response handling.
- **Retry:** Enable automatic retries with options for exponential or linear jitter backoff.
- **Caching:** Enable caching with set time to live.
- **Logging:** Easily log all requests for better visibility.
- **Performance Monitor:** Monitor and log responses exceeding defined thresholds.
- **Authentication:** utomatically include access tokens and refresh them in the background.

## Getting Started

### Installation

```shell
go get -u github.com/liviudnicoara/swiftreq
```

### Usage

Making simple requests

```go

    // GET request
    posts, err := swiftreq.Get[[]Post](BASE_URL + "/posts").Do(context.Background())
	
	// POST request
	resp, err := swiftreq.Post[Post](BASE_URL+"/posts", post).Do(context.Background())
	
	// PUT request
	resp, err = swiftreq.Put[Post](BASE_URL+"/posts/1", post).Do(context.Background())
	
	// DELETE request
	resp, err = swiftreq.Delete[Post](BASE_URL + "/posts/1").Do(context.Background())
	
```

Making custom requests

```go

	post, err := swiftreq.Get[Post](BASE_URL + "/posts/1").
		WithQueryParameters(map[string]string{"page": "1"}).
		WithHeaders(map[string]string{"Content-Type": "application/json"}).
		Do(context.Background())

```

Setting retry

```go

    // Request with exponential backoff (default min wait is 500ms and max wait is 10s)
    resp, err := swiftreq.Get[string]("http://localhost:3000/retry").
		WithRequestExecutor(swiftreq.Default().
			WithExponentialRetry(5)).
		Do(context.Background())

    // Request with liniar backoff (default min wait is 500ms and max wait is 10s)
    resp, err := swiftreq.Get[string]("http://localhost:3000/retry").
		WithRequestExecutor(swiftreq.Default().
			WithLiniarRetry(5)).
		Do(context.Background())

```

Caching responses

```go
	swiftreq.Default(). 
		AddCaching(100 * time.Second) // Get requests will be cached in memory for 100 seconds

	post, err := swiftreq.Get[Post](BASE_URL + "/posts/1").
		Do(context.Background())

```

Logging and performance monitor

```go
	swiftreq.Default(). 
		AddLogging(slog.Default()).                                // add logger
		AddPerformanceMonitor(10*time.Millisecond, slog.Default()) // add performance monitor

    // Requests will be logged. 
    // If response time is over 10 ms, a warning will be logged.
	post, err := swiftreq.Get[Post](BASE_URL + "/posts/1").
		Do(context.Background())

```

Authentication

```go

    re := swiftreq.Default()
        .WithAuthorization("Token", func() (token string, lifeSpan time.Duration, err error) {
            // Provide the token retrieval.
            resp, err := swiftreq.Get[string]("http://localhost:3000/auth").
                WithRequestExecutor(swiftreq.NewRequestExecutor(*http.DefaultClient)).
                WithHeaders(map[string]string{"Credentials": "user:pass"}).
                Do(context.Background())

            return resp.Token resp.LifeSpan, resp.Error
	    })


	resp, err := swiftreq.Get[string]("http://localhost:3000/page").
		WithRequestExecutor(re).
		Do(context.Background())

```


## License
This project is licensed under the MIT License - see the [License](https://raw.githubusercontent.com/liviudnicoara/swiftreq/master/LICENSE) file for details.