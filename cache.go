package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var ctx = context.Background()

const prefix = "krakend"

type Cache struct {
	redis  *redis.Client
	logger Logger
}

func NewCache(redis *redis.Client, logger Logger) Cache {
	return Cache{redis, logger}
}

func (c *Cache) Set(url string, res *http.Response, ttl int) {
	body, err := httputil.DumpResponse(res, true)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Can't dump response. Url: %s. Error: %v", url, err))
		return
	}

	c.redis.Set(ctx, key(url), body, time.Duration(ttl)*time.Second)
}

func (c *Cache) Get(url string) *http.Response {
	v, err := c.redis.Get(ctx, key(url)).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil
		}

		c.logger.Error(fmt.Sprintf("Can't get response cache from redis. Url: %s", url))
		return nil
	}

	buf := bufio.NewReader(bytes.NewReader(v))
	r, err := http.ReadResponse(buf, nil)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Can't read from cache. Url: %s", url))
		return nil
	}

	return r
}

func key(url string) string {
	return fmt.Sprintf("%s:%s", prefix, uuid.NewSHA1(uuid.NameSpaceURL, []byte(url)).String())
}
