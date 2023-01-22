package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

var redisServer *miniredis.Miniredis
var h *CacheHandler
var body string = `{"message": "Hello Wordl"}`

// Empty logger implementation
type noopLogger struct{}

func (n noopLogger) Debug(_ ...interface{})    {}
func (n noopLogger) Info(_ ...interface{})     {}
func (n noopLogger) Warning(_ ...interface{})  {}
func (n noopLogger) Error(_ ...interface{})    {}
func (n noopLogger) Critical(_ ...interface{}) {}
func (n noopLogger) Fatal(_ ...interface{})    {}

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: redisServer.Addr(),
	})
}

func mockRedis() *miniredis.Miniredis {
	s, err := miniredis.Run()

	if err != nil {
		panic(err)
	}

	return s
}

func setup() {
	redisServer = mockRedis()
	httpmock.Activate()

	logger := noopLogger{}
	h = NewCacheHandler(*http.DefaultClient, NewCache(newRedisClient(), logger), logger)
}

func teardown() {
	redisServer.Close()
	httpmock.DeactivateAndReset()
}

func TestHandleNotSupportedMethods(t *testing.T) {
	setup()
	defer teardown()

	handler := h.NewHandler(10)

	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	}

	for _, method := range methods {
		req, _ := http.NewRequest(method, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Result().StatusCode, "Status code mismatch")
	}
}

func TestHandleSuccess(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder(http.MethodGet, "/test", httpmock.NewStringResponder(200, body))
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)

	rr := httptest.NewRecorder()
	handler := h.NewHandler(10)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, 200, rr.Result().StatusCode, "Status code mismatch")
	assert.Equal(t, body, rr.Body.String(), "Body mismatch")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, 200, rr.Result().StatusCode, "Status code mismatch")
	assert.Equal(t, body, rr.Body.String(), "Body mismatch")
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestHandleStaleCache(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder(http.MethodGet, "/test", httpmock.NewStringResponder(200, body))
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)

	rr := httptest.NewRecorder()
	handler := h.NewHandler(10)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, 200, rr.Result().StatusCode, "Status code mismatch")
	assert.Equal(t, body, rr.Body.String(), "Body mismatch")

	redisServer.FastForward(time.Duration(10) * time.Second)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, 200, rr.Result().StatusCode, "Status code mismatch")
	assert.Equal(t, body, rr.Body.String(), "Body mismatch")
	assert.Equal(t, 2, httpmock.GetTotalCallCount())
}

func TestHandleNotSuccessResponse(t *testing.T) {
	setup()
	defer teardown()

	statuses := []int{
		100,
		300,
		400,
		404,
		500,
		501,
	}

	for _, status := range statuses {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		httpmock.RegisterResponder(http.MethodGet, "/test", httpmock.NewStringResponder(status, body))

		rr := httptest.NewRecorder()
		handler := h.NewHandler(10)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, status, rr.Result().StatusCode, "Status code mismatch")
		assert.Equal(t, body, rr.Body.String(), "Body mismatch")
		assert.Equal(t, "", redisServer.Dump())
	}
}
