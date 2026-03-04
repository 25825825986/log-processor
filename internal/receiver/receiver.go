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

// Handler 数据处理函数类型
type Handler func(line string)

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
		httpReceiver := NewHTTPReceiver(m.config.HTTPPort)
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
			r.handler(line)
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
			r.handler(line)
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
	port    int
	server  *http.Server
	handler Handler
}

// NewHTTPReceiver 创建HTTP接收器
func NewHTTPReceiver(port int) *HTTPReceiver {
	return &HTTPReceiver{
		port: port,
	}
}

// Start 启动HTTP接收器
func (r *HTTPReceiver) Start(handler Handler) error {
	r.handler = handler

	mux := http.NewServeMux()
	mux.HandleFunc("/logs", r.handleLogs)
	mux.HandleFunc("/health", r.handleHealth)

	r.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", r.port),
		Handler: mux,
	}

	log.Printf("HTTP receiver listening on port %d", r.port)
	return r.server.ListenAndServe()
}

// handleLogs 处理日志提交
func (r *HTTPReceiver) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			r.handler(line)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
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

// ImportFile 导入文件
func (f *FileImporter) ImportFile(filepath string, handler Handler) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
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
		return err
	}

	log.Printf("Imported %d lines from %s", lineCount, filepath)
	return nil
}


