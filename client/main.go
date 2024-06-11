package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ip                    = flag.String("ip", "localhost", "server IP")
	connections           = flag.Int("conn", 1, "number of websocket connections")
	successfulConnections int32
	messagesSent          int32
	latencies             []time.Duration
	mu                    sync.Mutex
)

func recordLatency(start time.Time) {
	latency := time.Since(start)
	mu.Lock()
	latencies = append(latencies, latency)
	mu.Unlock()
}

func calculateLatencyStats() {
	mu.Lock()
	defer mu.Unlock()

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	n := len(latencies)
	if n == 0 {
		log.Println("No latencies recorded.")
		return
	}

	sum := time.Duration(0)
	for _, latency := range latencies {
		sum += latency
	}

	mean := sum / time.Duration(n)
	median := latencies[n/2]
	p90 := latencies[int(float64(n)*0.9)]
	p99 := latencies[int(float64(n)*0.99)]
	max := latencies[n-1]

	log.Printf("Latency stats - Mean: %v, 50th Percentile: %v, 90th Percentile: %v, 99th Percentile: %v, Max: %v",
		mean, median, p90, p99, max)
}

func main() {
	flag.Usage = func() {
		io.WriteString(os.Stderr, `Websockets client generator Example usage: ./client -ip=172.17.0.1 -conn=10`)
		flag.PrintDefaults()
	}
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *ip + ":9000", Path: "/"}
	log.Printf("Connecting to %s", u.String())

	var conns []*websocket.Conn
	for i := 0; i < *connections; i++ {
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Println("Failed to connect", i, err)
			continue
		}
		atomic.AddInt32(&successfulConnections, 1)
		conns = append(conns, c)
		defer func(c *websocket.Conn) {
			c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			time.Sleep(time.Second)
			c.Close()
		}(c)
	}

	log.Printf("Finished initializing %d connections", len(conns))

	var wg sync.WaitGroup
	ticker := time.NewTicker(1 * time.Second)
	reportTicker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			for i := 0; i < len(conns); i++ {
				wg.Add(1)
				go func(conn *websocket.Conn, i int) {
					defer wg.Done()
					start := time.Now()
					//log.Printf("Conn %d sending message", i)
					if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second*5)); err != nil {
						fmt.Printf("Failed to receive pong: %v", err)
					}
					err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Hello from conn %v", i)))
					if err == nil {
						atomic.AddInt32(&messagesSent, 1)
						recordLatency(start)
					}
				}(conns[i], i)
			}
			wg.Wait()
			//log.Printf("Successful connections: %d, Messages sent: %d", atomic.LoadInt32(&successfulConnections), atomic.LoadInt32(&messagesSent))
		case <-reportTicker.C:
			calculateLatencyStats()
		}
	}
}
