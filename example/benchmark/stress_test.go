// 并发性能测试工具
package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	protocol    = flag.String("protocol", "tcp", "协议类型: tcp, udp, http")
	addr        = flag.String("addr", "localhost:9000", "目标地址")
	total       = flag.Int("total", 10000, "总发送日志数")
	concurrency = flag.Int("c", 100, "并发连接数/协程数")
	duration    = flag.Duration("d", 0, "测试持续时间(如30s)，0表示按total发送")
	batchSize   = flag.Int("batch", 1, "每批发送条数(仅HTTP有效)")
)

type Stats struct {
	sent      int64
	failed    int64
	startTime time.Time
}

func main() {
	flag.Parse()

	stats := &Stats{startTime: time.Now()}

	fmt.Printf("=== 日志处理器并发测试 ===\n")
	fmt.Printf("协议: %s\n", *protocol)
	fmt.Printf("目标: %s\n", *addr)
	fmt.Printf("并发: %d\n", *concurrency)
	if *duration > 0 {
		fmt.Printf("持续时间: %s\n", *duration)
	} else {
		fmt.Printf("总量: %d\n", *total)
	}
	fmt.Println()

	// 启动进度报告
	stopReport := make(chan bool)
	go reportProgress(stats, stopReport)

	// 执行测试
	var wg sync.WaitGroup
	
	switch *protocol {
	case "tcp":
		runTCPSender(&wg, stats)
	case "udp":
		runUDPSender(&wg, stats)
	case "http":
		runHTTPSender(&wg, stats)
	default:
		fmt.Printf("未知协议: %s\n", *protocol)
		return
	}

	wg.Wait()
	close(stopReport)

	// 最终报告
	printFinalReport(stats)
}

func runTCPSender(wg *sync.WaitGroup, stats *Stats) {
	var count int64 = 0
	var shouldStop int32 = 0
	
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			conn, err := net.Dial("tcp", *addr)
			if err != nil {
				fmt.Printf("连接失败: %v\n", err)
				return
			}
			defer conn.Close()

			for {
				if *duration > 0 {
					if time.Since(stats.startTime) > *duration {
						return
					}
				} else {
					if atomic.AddInt64(&count, 1) > int64(*total) {
						atomic.AddInt64(&count, -1)
						return
					}
				}
				
				if atomic.LoadInt32(&shouldStop) == 1 {
					return
				}

				log := generateLogLine(int(atomic.LoadInt64(&count)))
				_, err := conn.Write([]byte(log + "\n"))
				if err != nil {
					atomic.AddInt64(&stats.failed, 1)
					// 重新连接
					conn.Close()
					conn, _ = net.Dial("tcp", *addr)
					if conn == nil {
						return
					}
				} else {
					atomic.AddInt64(&stats.sent, 1)
				}
			}
		}(i)
	}
}

func runUDPSender(wg *sync.WaitGroup, stats *Stats) {
	var count int64 = 0
	
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			conn, err := net.Dial("udp", *addr)
			if err != nil {
				fmt.Printf("连接失败: %v\n", err)
				return
			}
			defer conn.Close()

			for {
				if *duration > 0 {
					if time.Since(stats.startTime) > *duration {
						return
					}
				} else {
					if atomic.AddInt64(&count, 1) > int64(*total) {
						atomic.AddInt64(&count, -1)
						return
					}
				}

				log := generateLogLine(int(atomic.LoadInt64(&count)))
				_, err := conn.Write([]byte(log))
				if err != nil {
					atomic.AddInt64(&stats.failed, 1)
				} else {
					atomic.AddInt64(&stats.sent, 1)
				}
			}
		}(i)
	}
}

func runHTTPSender(wg *sync.WaitGroup, stats *Stats) {
	var count int64 = 0
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency,
			MaxIdleConnsPerHost: *concurrency,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	
	url := fmt.Sprintf("http://%s/logs", *addr)
	
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for {
				if *duration > 0 {
					if time.Since(stats.startTime) > *duration {
						return
					}
				} else {
					current := atomic.AddInt64(&count, int64(*batchSize))
					if current > int64(*total) {
						return
					}
				}

				var body string
				if *batchSize > 1 {
					var buf bytes.Buffer
					for j := 0; j < *batchSize; j++ {
						if j > 0 {
							buf.WriteByte('\n')
						}
						buf.WriteString(generateLogLine(int(atomic.LoadInt64(&count)) + j))
					}
					body = buf.String()
				} else {
					body = generateLogLine(int(atomic.LoadInt64(&count)))
				}

				resp, err := client.Post(url, "text/plain", bytes.NewBufferString(body))
				if err != nil {
					atomic.AddInt64(&stats.failed, int64(*batchSize))
					continue
				}
				resp.Body.Close()
				
				if resp.StatusCode == 200 {
					atomic.AddInt64(&stats.sent, int64(*batchSize))
				} else {
					atomic.AddInt64(&stats.failed, int64(*batchSize))
				}
			}
		}(i)
	}
}

func generateLogLine(id int) string {
	timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")
	path := fmt.Sprintf("/api/test%d", id%100)
	size := 100 + (id % 9900)
	return fmt.Sprintf(
		`127.0.0.1 - - [%s] "GET %s HTTP/1.1" 200 %d "-" "Benchmark/%d"`,
		timestamp, path, size, id,
	)
}

func reportProgress(stats *Stats, stop chan bool) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastSent int64 = 0
	
	for {
		select {
		case <-ticker.C:
			sent := atomic.LoadInt64(&stats.sent)
			duration := time.Since(stats.startTime).Seconds()
			qps := float64(sent-lastSent) / 1.0
			avgQPS := float64(sent) / duration
			
			fmt.Printf("\r[QPS: %6.0f/s] [Total: %8d] [Avg: %6.0f/s] [Failed: %d]        ",
				qps, sent, avgQPS, atomic.LoadInt64(&stats.failed))
			
			lastSent = sent
			
		case <-stop:
			fmt.Println()
			return
		}
	}
}

func printFinalReport(stats *Stats) {
	duration := time.Since(stats.startTime).Seconds()
	sent := atomic.LoadInt64(&stats.sent)
	failed := atomic.LoadInt64(&stats.failed)
	
	fmt.Println()
	fmt.Println("=== 测试结果 ===")
	fmt.Printf("总用时: %.2f 秒\n", duration)
	fmt.Printf("成功发送: %d 条\n", sent)
	fmt.Printf("失败: %d 条\n", failed)
	fmt.Printf("平均 QPS: %.0f 条/秒\n", float64(sent)/duration)
	fmt.Printf("吞吐量: %.2f MB/秒\n", float64(sent*100)/1024/1024/duration)
}
