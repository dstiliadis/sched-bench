package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/namsral/flag"
	"golang.org/x/sync/errgroup"
	"gonum.org/v1/gonum/stat/distuv"

	_ "net/http/pprof"
)

type statsInfo struct {
	OnTime        int64
	OffTime       int64
	Requests      int
	ExecutionTime int64
}

func main() {

	// Simple API client that super-imposes mutliple independent Markov ON/OFF
	// sources in order to emulate a realistic traffic scenario

	var url string
	var threads int
	var on, off float64
	var duration time.Duration

	flag.StringVar(&url, "url", "http://localhost:80/admin/test", "Target URL for requests")
	flag.IntVar(&threads, "threads", 1, "Number of parallel Go-Routines")
	flag.Float64Var(&on, "on", 0.3, "traffic ON rate")
	flag.Float64Var(&off, "off", 0.8, "traffic OFF rate")
	flag.DurationVar(&duration, "duration", 30*time.Second, "Time to run simulation")
	flag.Parse()

	fmt.Println("Running at rate per thread of", (1.0/on)/(1/on+1/off))

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	g := errgroup.Group{}

	stats := make([]*statsInfo, threads)
	start := time.Now()

	for i := 0; i < threads; i++ {
		s := &statsInfo{}
		stats[i] = s
		g.Go(func() error {
			return runClient(ctx, on, off, url, s)
		})
	}

	err := g.Wait()
	if err != nil {
		fmt.Println("Error in parallel threads", err)
		os.Exit(1)
	}

	total := &statsInfo{}
	for _, s := range stats {
		total.OnTime += s.OnTime
		total.OffTime += s.OffTime
		total.Requests += s.Requests
		total.ExecutionTime += s.ExecutionTime
	}

	fmt.Printf("Average API latency of API calls: %fus\n", float64(total.ExecutionTime)/float64(total.Requests)*1000)
	fmt.Println("Total Requests:", total.Requests)
	fmt.Println("Total Rate of API calls:", float64(total.Requests)/time.Since(start).Seconds())
	fmt.Println("Average On Time:", float64(total.OnTime)/float64(threads)/1000000.0)
	fmt.Println("Average Off Time", float64(total.OffTime)/float64(threads)/1000000.0)

	// Wait for ever to control Kubernetes results
	for {
		time.Sleep(100 * time.Second)
	}

}

func runClient(ctx context.Context, on, off float64, url string, stats *statsInfo) error {
	distOn := distuv.Exponential{
		Rate: on,
	}

	distOff := distuv.Exponential{
		Rate: off,
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			MaxIdleConns:        100,
			MaxConnsPerHost:     100,
			DisableCompression:  false,
			DisableKeepAlives:   false,
			TLSNextProto:        make(map[string]func(string, *tls.Conn) http.RoundTripper),
		},
		Timeout: 120 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:

			// Run for on-time
			onTime := time.Duration(distOn.Rand() * 10000000.0)
			requests, requestTime, err := doRequests(client, onTime, url)
			if err != nil {
				return err
			}

			// collect stats
			stats.OnTime += int64(onTime.Nanoseconds())
			stats.Requests += requests
			stats.ExecutionTime += requestTime

			// Sleep for off time
			offTime := time.Duration(distOff.Rand() * 10000000.0)
			stats.OffTime += int64(offTime.Nanoseconds())
			time.Sleep(offTime)
		}
	}
}

func doRequests(client *http.Client, onTime time.Duration, url string) (int, int64, error) {

	deadline := time.Now().Add(onTime)

	requests := 0
	var totalTime int64

	// Run until expiration time.
	for !time.Now().After(deadline) {

		requests++
		requestStart := time.Now()

		resp, err := client.Get(url)
		if err != nil {
			fmt.Println("Error from server", err)
			return -1, 0, err
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()

		totalTime = totalTime + time.Since(requestStart).Milliseconds()

	}

	return requests, totalTime, nil
}
