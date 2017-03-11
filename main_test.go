package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	provider = map[string][]int{
		"u=http://localhost:8090/primes&u=http://localhost:8090/fibo":                                    []int{1, 2, 3, 5, 7, 8, 11, 13, 21},
		"u=http://localhost:8090/fibo":                                                                   []int{1, 2, 3, 5, 8, 13, 21},
		"u=http://localhost:8090/rand":                                                                   []int{1, 3, 5, 7, 8, 10, 17, 19, 24, 27, 34, 76},
		"u=http://localhost:8090/rand&u=http://localhost:8090/odd":                                       []int{1, 3, 5, 7, 8, 9, 10, 11, 13, 15, 17, 19, 21, 23, 24, 27, 34, 76},
		"u=http://localhost:8090/rand&u=http://localhost:8090/odd&u=http://localhost:8090/fibo":          []int{1, 2, 3, 5, 7, 8, 9, 10, 11, 13, 15, 17, 19, 21, 23, 24, 27, 34, 76},
		"u=http://localhost:8090/fibo&u=http://localhost:8090/unavailable":                               []int{1, 2, 3, 5, 8, 13, 21},
		"u=http://localhost:8090/unavailable":                                                            []int{},
		"u=http://localhost:8090/slow/600":                                                               []int{},
		"u=http://localhost:8090/slow/500":                                                               []int{},
		"u=http://localhost:8090/slow/500&u=http://localhost:8090/slow/600":                              []int{},
		"u=http://localhost:8090/slow/500&u=http://localhost:8090/slow/600&u=http://localhost:8090/fibo": []int{1, 2, 3, 5, 8, 13, 21},
		"u=http://localhost:8090/slow/400":                                                               []int{100500},
		"u=http://localhost:8090/slow/400&u=http://localhost:8090/rand":                                  []int{1, 3, 5, 7, 8, 10, 17, 19, 24, 27, 34, 76, 100500},
		"u=http://localhost:8090/10-15&u=http://localhost:8090/20-25":                                    []int{10, 11, 12, 13, 14, 15, 20, 21, 22, 23, 24, 25},
		"u=http://localhost:8090/20-25&u=http://localhost:8090/10-15":                                    []int{10, 11, 12, 13, 14, 15, 20, 21, 22, 23, 24, 25},
		"u=broken": []int{},
	}
)

// TestMerge takes fixtures from provider
// and compares server answer with expected value
func TestMerge(t *testing.T) {
	for urls, expected := range provider {
		req := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/numbers?%s", urls), nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("unexpected status: %d", w.Code)
		}

		var numbers map[string][]int
		if err := json.NewDecoder(w.Body).Decode(&numbers); err != nil {
			t.Errorf("json decode: %s", err)
		}

		if fmt.Sprint(numbers["numbers"]) != fmt.Sprint(expected) {
			t.Errorf("invalid answer (%s): %v, expected: %v", urls, numbers["numbers"], expected)
		}
	}
}

func BenchmarkMerge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/numbers?u=localhost:8090/primes&u=localhost:8090/rand"), nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

// init stub server for testing
func init() {
	listenAddr := flag.String("http.addr", ":8090", "http listen address")
	flag.Parse()

	http.HandleFunc("/primes", numbers([]int{2, 3, 5, 7, 11, 13}))
	http.HandleFunc("/fibo", numbers([]int{1, 1, 2, 3, 5, 8, 13, 21}))
	http.HandleFunc("/odd", numbers([]int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23}))
	http.HandleFunc("/rand", numbers([]int{5, 17, 3, 19, 76, 24, 1, 5, 10, 34, 8, 27, 7}))
	http.HandleFunc("/10-15", numbers([]int{10, 11, 12, 13, 14, 15}))
	http.HandleFunc("/20-25", numbers([]int{20, 21, 22, 23, 24, 25}))
	http.HandleFunc("/unavailable", unavailable())
	http.HandleFunc("/slow/400", slow(time.Millisecond*400))
	http.HandleFunc("/slow/500", slow(time.Millisecond*500))
	http.HandleFunc("/slow/600", slow(time.Millisecond*600))

	go http.ListenAndServe(*listenAddr, nil)
}

func numbers(numbers []int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]interface{}{"numbers": numbers})
	}
}

func unavailable() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
}

func slow(period time.Duration) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		time.Sleep(period)

		json.NewEncoder(w).Encode(map[string]interface{}{"numbers": []int{100500}})
	}
}
