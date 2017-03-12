package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
	"fmt"
)

const defaultTimeout = 500

// do not need keep alive if hosts always different
const defaultKeepAlive = 0

var (
	timeout = time.Millisecond * defaultTimeout
	ctx     = context.Background()
	client  http.Client
)

// NumbersResponse format
type NumbersResponse struct {
	Numbers []int `json:"numbers"`
}

func main() {
	addr := flag.String("addr", ":8080", "server listen address")
	tmt := flag.Int("timeout", defaultTimeout, "request timeout in milliseconds (to the urls)")
	timeout = time.Duration(*tmt) * time.Millisecond

	flag.Parse()

	client = http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: defaultKeepAlive,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: timeout,
	}

	s := &http.Server{
		Addr:           *addr,
		Handler:        http.HandlerFunc(handler),
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
		MaxHeaderBytes: 1 << 10,
	}

	log.Printf("Listen on %s", *addr)
	log.Fatal(s.ListenAndServe())
}

// handler listens all requests
// if URI is not "numbers" returns 404
// else merges numbers getting from other
// sites transferred by query param u.
func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/numbers" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Init
	var (
		urls        []string
		tube        = make(chan []int)
		ctx, cancel = context.WithCancel(ctx)
	)

	// Timer started at the top
	timer := time.AfterFunc(timeout, func() {
		log.Print("Timeout, close all requests")
		cancel()
	})
	defer timer.Stop()

	// URL decoded yet
	urls = r.URL.Query()["u"]
	if len(urls) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Async requests
	wg := sync.WaitGroup{}
	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			// If error, only log it
			numbers, err := receiveNumbers(url, ctx)
			if err != nil {
				log.Printf("Background err: %s", err)
				return
			}

			// Send numbers to the merge
			tube <- numbers
		}(u)
	}

	// Waiting results
	go func() {
		wg.Wait()
		close(tube)
	}()

	// Listen channel of numbers
	var sset []int
	for new := range tube {

		// Nothing to do
		if len(new) == 0 {
			continue
		}

		// Sort new data before merging
		sort.Ints(new)

		// First iteration
		if len(sset) == 0 {
			sset = uniqFromSorted(new)

			continue
		}

		// Append to the tail
		if sset[len(sset)-1] < new[0] {
			sset = append(sset, uniqFromSorted(new)...)
			continue
		}

		// Append to the head
		if new[len(new)-1] < sset[0] {
			sset = append(uniqFromSorted(new), sset...)
			continue
		}

		// Merge
		merged := make([]int, 0, len(new)+len(sset))
		p := 0
		l := 0
		for k, e := range sset {

			if l == e && k != 0 {
				continue
			}

			for _, n := range new[p:] {
				if e <= n {
					if e == n {
						p++
					}
					l = e
					k++
					merged = append(merged, l)
					break
				} else {
					p++
					if e == n || l == n {
						continue
					}

					l = n
					merged = append(merged, l)
				}
			}

			if p == len(new) {
				merged = append(merged, sset[k:]...)
				break
			}
		}

		merged = append(merged, uniqFromSorted(new[p:])...)

		sset = merged
	}

	// Write response
	json.NewEncoder(w).Encode(NumbersResponse{Numbers: sset})
}

// uniqFromSorted takes sorted slice and returns it without duplicates
func uniqFromSorted(arr []int) []int {
	var (
		last int
		res  = make([]int, 0, len(arr))
	)

	for k, v := range arr {
		if k != 0 && v == last {
			continue
		}
		last = v
		res = append(res, v)
	}

	return res
}

// receiveNumbers from remote server
func receiveNumbers(url string, ctx context.Context) ([]int, error) {
	var (
		numbers NumbersResponse
		err     error
		res     *http.Response
		req     *http.Request
	)

	// New request will be closed if it takes more than timeout value
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("new number request: %s", err)
	}
	req = req.WithContext(ctx)

	res, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("during number request err: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returns unexpected ststus: %d", res.StatusCode)
	}

	dec := json.NewDecoder(res.Body)
	if err = dec.Decode(&numbers); err != nil {
		return nil, fmt.Errorf("decode numbers response: %s", err)
	}

	return numbers.Numbers, nil
}
