package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Router struct {
	edges     []*url.URL
	regionMap map[string]*url.URL
	current   uint64
	healthy   map[string]bool
	mu        sync.RWMutex
}

func NewRouter() *Router {
	edgeMap := os.Getenv("EDGE_MAP")
	regionMap := make(map[string]*url.URL)
	edges := make([]*url.URL, 0)

	for pair := range strings.SplitSeq(edgeMap, ",") {
		parts := strings.SplitN(pair, "=", 2)
		region := parts[0]
		u, err := url.Parse(parts[1])
		if err != nil {
			log.Fatalf("invalid edge URL for region %s: %v", region, err)
		}
		regionMap[region] = u
		edges = append(edges, u)
	}

	router := &Router{
		edges:     edges,
		regionMap: regionMap,
		healthy: make(map[string]bool),
	}

	router.StartHealthChecks(3 * time.Second)

	return router
}

func (r *Router) checkHealth(edge *url.URL) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(edge.String() + "/health")
	if err != nil {
		return false
	}

	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (r *Router) isHealthy(edge *url.URL) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status, exists := r.healthy[edge.String()]
	return !exists || status
}

func (r *Router) StartHealthChecks(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			for _, edge := range r.edges {
				status := r.checkHealth(edge)
				r.mu.Lock()
				r.healthy[edge.String()] = status
				r.mu.Unlock()

				if !status {
					log.Printf("[HEALTH] %s is down", edge.Host)
				}
			}
		}
	}()
}

func (r *Router) Next() *url.URL {
	for i := 0; i < len(r.edges); i++ {
		idx := atomic.AddUint64(&r.current, 1)
		edge := r.edges[idx%uint64(len(r.edges))]
		if r.isHealthy(edge) {
			return edge
		}
	}

	return r.edges[0]
}

func (r *Router) PickEdge(req *http.Request) *url.URL {
	region := req.Header.Get("X-Region")

	if edge, ok := r.regionMap[region]; ok && r.isHealthy(edge) {
		return edge
	}

	return r.Next()
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	target := r.PickEdge(req)

	proxy := httputil.NewSingleHostReverseProxy(target)
	log.Printf("[ROUTER] %s %s → %s", req.Method, req.URL.Path, target.Host)
	proxy.ServeHTTP(w, req)
}
