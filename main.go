package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type registerer string

var ClientRegisterer = registerer("krakend-http-cache-redis")
var logger Logger = nil
var handler *CacheHandler

func (registerer) RegisterLogger(v interface{}) {
	l, ok := v.(Logger)
	if !ok {
		return
	}
	logger = l
}

func (r registerer) RegisterClients(f func(
	name string,
	handler func(context.Context, map[string]interface{}) (http.Handler, error),
)) {
	addr := os.Getenv("KRAKEND_REDIS_CACHE_ADDR")
	if addr == "" {
		logger.Error("KRAKEND_REDIS_CACHE_ADDR is empty")
		return
	}

	db, err := strconv.Atoi(os.Getenv("KRAKEND_REDIS_CACHE_DB"))
	if err != nil {
		db = 0
	}

	poolSize, err := strconv.Atoi(os.Getenv("KRAKEND_REDIS_CACHE_POOL_SIZE"))
	if err != nil {
		poolSize = 10
	}

	redis := redis.NewClient(&redis.Options{
		Addr:       addr,
		DB:         db,
		MaxRetries: 1,
		PoolSize:   poolSize,
	})
	handler = NewCacheHandler(*http.DefaultClient, NewCache(redis, logger), logger)
	f(string(r), r.registerClients)
}

func (r registerer) registerClients(_ context.Context, extra map[string]interface{}) (http.Handler, error) {
	name, ok := extra["name"].(string)
	if !ok {
		return nil, errors.New("wrong config")
	}
	if name != string(r) {
		return nil, fmt.Errorf("unknown register %s", name)
	}

	config, _ := extra[string(r)].(map[string]interface{})

	cache_ttl, ok := config["cache_ttl"].(float64)
	if !ok {
		return nil, errors.New("wrong cache ttl")
	}

	return handler.NewHandler(int(cache_ttl)), nil
}

func main() {}
