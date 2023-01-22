package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

type CacheHandler struct {
	client http.Client
	cache  Cache
	logger Logger
}

func NewCacheHandler(client http.Client, cache Cache, logger Logger) *CacheHandler {
	return &CacheHandler{client, cache, logger}
}

func (h *CacheHandler) NewHandler(ttl int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		url := req.URL.RequestURI()

		if req.Method != http.MethodGet {
			h.logger.Error(fmt.Sprintf("Can't cache not GET method. URL: %s. Method: %s", url, req.Method))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		res := h.cache.Get(url)
		if res != nil {
			writeResponse(w, res)
			return
		}

		h.makeRealRequest(w, req, ttl)
	})
}

func (h *CacheHandler) makeRealRequest(w http.ResponseWriter, req *http.Request, ttl int) {
	res, err := h.client.Do(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeResponse(w, res)
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		h.cache.Set(req.URL.RequestURI(), res, ttl)
	}
}

func writeResponse(w http.ResponseWriter, res *http.Response) {
	defer res.Body.Close()

	for k, hs := range res.Header {
		for _, h := range hs {
			w.Header().Add(k, h)
		}
	}

	w.WriteHeader(res.StatusCode)
	if res.Body == nil {
		return
	}

	bodyBytes, _ := ioutil.ReadAll(res.Body)
	_, _ = w.Write(bodyBytes)
	res.Body.Close()
	res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
}
