### ENV Vars

`KRAKEND_REDIS_CACHE_ADDR` - Redis address (Example: `0.0.0.0:6379`)

`KRAKEND_REDIS_CACHE_DB` - Redis DB (Default: 0)

`KRAKEND_REDIS_CACHE_POOL_SIZE` - Redis client pool size (Default: 10)

### Endpoints Config Example

```json
{
  "version": 3,
  "name": "KrakenD Enterprise API Gateway",
  "plugin": {
    "pattern": ".so",
    "folder": "/etc/krakend/plugins"
  },
  "endpoints": [
    {
      "endpoint": "/hello",
      "backend": [
        {
          "host": ["http://api:8080"],
          "url_pattern": "/hello",
          "extra_config": {
            "plugin/http-client": {
              "name": "krakend-http-cache-redis",
              "krakend-http-cache-redis": {
                "cache_ttl": 30
              }
            }
          }
        }
      ]
    }
  ]
}
```
