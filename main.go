package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

func main() {
	typeArg := flag.String("type", "default", "the type to use")
	locationArg := flag.String("location", "default", "edge node location")
	portArg := flag.String("port", "default", "edge node port")

	flag.Parse()

	switch *typeArg {
	case "origin":
		startOrigin()
	case "node":
		cache := NewCache()
		startNode(*locationArg, *portArg, cache)
	case "router":
		startRouter()
	}
}

func startOrigin() {
	fmt.Println("Starting origin")

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from origin")
	})

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func startRouter() {
	fmt.Println("Starting router in por 8081")

	r := NewRouter()

	err := http.ListenAndServe(":8081", r)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func startNode(location string, port string, cache *Cache) {
	fmt.Println("Starting node in", location, "on port", port)

	originAddr := os.Getenv("ORIGIN_URL")
	if originAddr == "" {
		log.Fatal("No origin URL")
	}

	originURL, err := url.Parse(originAddr)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(originURL)

	proxy.ModifyResponse = func(resp *http.Response) error {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		cache.Set(resp.Request.URL.Path, &CacheEntry{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header.Clone(),
			Body:       body,
			CreatedAt:  time.Now(),
			TTL:        5 * time.Second,
		})

		resp.Body = io.NopCloser(bytes.NewReader(body))

		return nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		if entry, hit := cache.Get(r.URL.Path); hit {
			log.Printf("[CACHE HIT] %s", r.URL.Path)
			for k, v := range entry.Headers {
				w.Header()[k] = v
			}
			w.Header().Set("X-Cache-Status", "HIT")
			w.WriteHeader(entry.StatusCode)
			w.Write(entry.Body)
			return
		}

		log.Printf("[CACHE MISS] %s → proxying to origin", r.URL.Path)
		w.Header().Set("X-Cache-Status", "MISS")
		proxy.ServeHTTP(w, r)
	})

	http.ListenAndServe(":"+port, handler)
}
