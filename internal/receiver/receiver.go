// receiver/receiver.go - 数据接收器
package receiver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"log-processor/internal/config"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Handler 数据处理函数类型，返回是否成功处理
type Handler func(line string) bool

// Receiver 接收器接口
type Receiver interface {
	Start(handler Handler) error
	Stop() error
}

// Manager 接收器管理器
type Manager struct {
	config    config.ReceiverConfig
	handler   Handler
	receivers []Receiver
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewManager 创建接收器管理器
func NewManager(cfg config.ReceiverConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		receivers: make([]Receiver, 0),
	}
}

// Start 启动所有接收器
func (m *Manager) Start(handler Handler) error {
	m.handler = handler

	if m.config.TCPEnabled {
		tcpReceiver := NewTCPReceiver(m.config.TCPPort, m.config.BufferSize)
		m.receivers = append(m.receivers, tcpReceiver)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := tcpReceiver.Start(m.handler); err != nil {
				log.Printf("TCP receiver error: %v", err)
			}
		}()
	}

	if m.config.UDPEnabled {
		udpReceiver := NewUDPReceiver(m.config.UDPPort, m.config.BufferSize)
		m.receivers = append(m.receivers, udpReceiver)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := udpReceiver.Start(m.handler); err != nil {
				log.Printf("UDP receiver error: %v", err)
			}
		}()
	}

	if m.config.HTTPEnabled {
		httpReceiver := NewHTTPReceiver(
			m.config.HTTPPort,
			m.config.HTTPAuthToken,
			m.config.HTTPAllowedIPs,
			m.config.HTTPMaxBodySize,
			m.config.HTTPRateLimit,
		)
		m.receivers = append(m.receivers, httpReceiver)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := httpReceiver.Start(m.handler); err != nil {
				log.Printf("HTTP receiver error: %v", err)
			}
		}()
	}

	return nil
}

// Stop 停止所有接收器
func (m *Manager) Stop() error {
	m.cancel()
	for _, r := range m.receivers {
		r.Stop()
	}
	m.wg.Wait()
	return nil
}

// TCPReceiver TCP接收器
type TCPReceiver struct {
	port      int
	bufferSize int
	listener  net.Listener
	handler   Handler
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewTCPReceiver 创建TCP接收器
func NewTCPReceiver(port, bufferSize int) *TCPReceiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPReceiver{
		port:       port,
		bufferSize: bufferSize,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动TCP接收器
func (r *TCPReceiver) Start(handler Handler) error {
	r.handler = handler

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.port))
	if err != nil {
		return err
	}
	r.listener = listener

	log.Printf("TCP receiver listening on port %d", r.port)

	for {
		select {
		case <-r.ctx.Done():
			return nil
		default:
		}

		listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))
		conn, err := listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			if r.ctx.Err() != nil {
				return nil
			}
			log.Printf("Accept error: %v", err)
			continue
		}

		go r.handleConnection(conn)
	}
}

// handleConnection 处理连接
func (r *TCPReceiver) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReaderSize(conn, r.bufferSize)
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			if !r.handler(line) {
				// 处理器队列满，数据已丢失
				// 短暂延迟让队列消化，但不重试（避免阻塞接收）
				time.Sleep(time.Millisecond * 10)
			}
		}
	}
}

// Stop 停止TCP接收器
func (r *TCPReceiver) Stop() error {
	r.cancel()
	if r.listener != nil {
		r.listener.Close()
	}
	return nil
}

// UDPReceiver UDP接收器
type UDPReceiver struct {
	port       int
	bufferSize int
	conn       *net.UDPConn
	handler    Handler
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewUDPReceiver 创建UDP接收器
func NewUDPReceiver(port, bufferSize int) *UDPReceiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &UDPReceiver{
		port:       port,
		bufferSize: bufferSize,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动UDP接收器
func (r *UDPReceiver) Start(handler Handler) error {
	r.handler = handler

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", r.port))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	r.conn = conn

	log.Printf("UDP receiver listening on port %d", r.port)

	buffer := make([]byte, r.bufferSize)
	for {
		select {
		case <-r.ctx.Done():
			return nil
		default:
		}

		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			if r.ctx.Err() != nil {
				return nil
			}
			log.Printf("UDP read error: %v", err)
			continue
		}

		line := strings.TrimSpace(string(buffer[:n]))
		if line != "" {
			if !r.handler(line) {
				// 处理器队列满，增加短暂延迟让队列消化
				time.Sleep(time.Millisecond)
			}
		}
	}
}

// Stop 停止UDP接收器
func (r *UDPReceiver) Stop() error {
	r.cancel()
	if r.conn != nil {
		r.conn.Close()
	}
	return nil
}

// HTTPReceiver HTTP接收器
type HTTPReceiver struct {
	port         int
	authToken    string
	allowedIPs   []string
	maxBodySize  int64
	rateLimit    int
	server       *http.Server
	handler      Handler
	requestCount map[string]int // IP -> 请求计数
	lastReset    time.Time
	mu           sync.Mutex
}

// NewHTTPReceiver 创建HTTP接收器
func NewHTTPReceiver(port int, authToken string, allowedIPs []string, maxBodySize int64, rateLimit int) *HTTPReceiver {
	if maxBodySize == 0 {
		maxBodySize = 10 * 1024 * 1024 // 默认10MB
	}
	return &HTTPReceiver{
		port:         port,
		authToken:    authToken,
		allowedIPs:   allowedIPs,
		maxBodySize:  maxBodySize,
		rateLimit:    rateLimit,
		requestCount: make(map[string]int),
		lastReset:    time.Now(),
	}
}

// Start 启动HTTP接收器
func (r *HTTPReceiver) Start(handler Handler) error {
	r.handler = handler

	mux := http.NewServeMux()
	mux.HandleFunc("/logs", r.handleLogs)
	mux.HandleFunc("/health", r.handleHealth)

	r.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", r.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("HTTP receiver listening on port %d", r.port)
	if r.authToken != "" {
		log.Printf("HTTP receiver auth token enabled")
	}
	if len(r.allowedIPs) > 0 {
		log.Printf("HTTP receiver IP whitelist: %v", r.allowedIPs)
	}
	if r.rateLimit > 0 {
		log.Printf("HTTP receiver rate limit: %d requests/min per IP", r.rateLimit)
	}

	err := r.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil // 正常关闭，不是错误
	}
	return err
}

// checkAuth 检查认证
func (r *HTTPReceiver) checkAuth(req *http.Request) bool {
	// 如果未配置Token，允许匿名访问（但不推荐）
	if r.authToken == "" {
		return true
	}

	// 从Header或Query参数获取Token
	token := req.Header.Get("X-Auth-Token")
	if token == "" {
		token = req.URL.Query().Get("token")
	}

	return token == r.authToken
}

// checkIPAllowed 检查IP白名单
func (r *HTTPReceiver) checkIPAllowed(req *http.Request) bool {
	if len(r.allowedIPs) == 0 {
		return true
	}

	clientIP := getClientIP(req)
	for _, ip := range r.allowedIPs {
		if ip == clientIP {
			return true
		}
	}
	return false
}

// checkRateLimit 检查速率限制
func (r *HTTPReceiver) checkRateLimit(req *http.Request) bool {
	if r.rateLimit <= 0 {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 每分钟重置计数器
	if time.Since(r.lastReset) > time.Minute {
		r.requestCount = make(map[string]int)
		r.lastReset = time.Now()
	}

	clientIP := getClientIP(req)
	r.requestCount[clientIP]++

	return r.requestCount[clientIP] <= r.rateLimit
}

// getClientIP 获取客户端真实IP
func getClientIP(req *http.Request) string {
	// 优先从X-Forwarded-For获取（代理场景）
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 从X-Real-Ip获取
	xri := req.Header.Get("X-Real-Ip")
	if xri != "" {
		return xri
	}

	// 直接连接IP
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	return host
}

// handleLogs 处理日志提交
func (r *HTTPReceiver) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 检查IP白名单
	if !r.checkIPAllowed(req) {
		log.Printf("[SECURITY] Blocked request from IP: %s (not in whitelist)", getClientIP(req))
		http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
		return
	}

	// 检查认证
	if !r.checkAuth(req) {
		log.Printf("[SECURITY] Unauthorized request from IP: %s", getClientIP(req))
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// 检查速率限制
	if !r.checkRateLimit(req) {
		log.Printf("[SECURITY] Rate limit exceeded for IP: %s", getClientIP(req))
		http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
		return
	}

	// 限制请求体大小
	req.Body = http.MaxBytesReader(w, req.Body, r.maxBodySize)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("[SECURITY] Request too large from IP: %s", getClientIP(req))
		http.Error(w, `{"error":"Request too large"}`, http.StatusRequestEntityTooLarge)
		return
	}
	defer req.Body.Close()

	// 限制处理行数，防止DoS
	lines := strings.Split(string(body), "\n")
	if len(lines) > 10000 {
		log.Printf("[SECURITY] Too many lines from IP: %s (%d lines)", getClientIP(req), len(lines))
		http.Error(w, `{"error":"Too many lines (max 10000)"}`, http.StatusBadRequest)
		return
	}

	processedCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			r.handler(line)
			processedCount++
		}
	}

	log.Printf("[HTTP] Processed %d lines from %s", processedCount, getClientIP(req))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status":"ok","processed":%d}`, processedCount)))
}

// handleHealth 健康检查
func (r *HTTPReceiver) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// Stop 停止HTTP接收器
func (r *HTTPReceiver) Stop() error {
	if r.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return r.server.Shutdown(ctx)
	}
	return nil
}

// FileImporter 文件导入器
type FileImporter struct {
	handler Handler
}

// NewFileImporter 创建文件导入器
func NewFileImporter() *FileImporter {
	return &FileImporter{}
}

// ImportFile 导入文件 (返回成功导入的行数)
func (f *FileImporter) ImportFile(filepath string, handler func(string) bool) (int, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			handler(line)
			lineCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return lineCount, err
	}

	log.Printf("Imported %d lines from %s", lineCount, filepath)
	return lineCount, nil
}


