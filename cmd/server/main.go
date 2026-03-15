// main.go - 程序入口
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/parser"
	"log-processor/internal/processor"
	"log-processor/internal/receiver"
	"log-processor/internal/server"
	"log-processor/internal/storage"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./config.json", "配置文件路径")
	flag.Parse()

	// 创建日志目录
	os.MkdirAll("./logs", 0755)
	
	// 设置日志文件
	logFileName := filepath.Join("./logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	defer logFile.Close()
	
	// 同时输出到终端和文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	
	log.Println("[START] 启动日志数据处理系统...")
	log.Printf("[INFO] 运行日志保存至: %s", logFileName)
	log.Printf("[TIP] 提示: 按 Ctrl+C 或输入 'exit' 停止服务")

	// 加载配置
	cfg := config.GetConfig()
	if err := cfg.LoadFromFile(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建数据目录
	os.MkdirAll("./data", 0755)
	os.MkdirAll("./exports", 0755)
	os.MkdirAll("./temp", 0755)

	// 初始化存储
	sqliteStore, err := storage.NewSQLiteStorage(cfg.Get().Storage)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}
	
	// 使用异步存储包装器，最大化 SQLite 单线程性能
	// buffer=50000: 缓冲5万条日志
	// batch=1000: 每批写入1000条
	// interval=100ms: 最长100ms刷新一次
	store := storage.NewAsyncStorage(sqliteStore, 50000, 1000, 100*time.Millisecond)
	log.Println("[INFO] 启用异步存储模式，写入缓冲: 50000条")

	// 初始化解析器
	parserCfg := cfg.GetParserConfig()
	logParser := parser.NewLogParser(parserCfg)

	// 初始化处理器
	processorCfg := cfg.GetProcessorConfig()
	proc := processor.NewProcessor(processorCfg, logParser, store)
	proc.Start()

	// 初始化接收器
	receiverCfg := cfg.GetReceiverConfig()
	recvManager := receiver.NewManager(receiverCfg)

	// 启动接收器
	err = recvManager.Start(func(line string) bool {
		// 提交到处理器
		if !proc.Submit(line) {
			log.Printf("处理器队列已满，丢弃日志: %s", line[:min(50, len(line))])
			return false
		}
		return true
	})
	if err != nil {
		log.Fatalf("启动接收器失败: %v", err)
	}

	// 初始化Web服务器
	srv := server.NewServer(cfg, store, proc, recvManager, logFile, configPath)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动服务器
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Run(); err != nil {
			serverErr <- err
		}
	}()

	log.Println("[OK] 系统启动成功！")
	
	// 自动打开浏览器
	webURL := fmt.Sprintf("http://localhost:%d", cfg.Get().Server.Port)
	log.Printf("[WEB] 正在打开浏览器: %s", webURL)
	go func() {
		// 等待一下确保服务器完全启动
		time.Sleep(500 * time.Millisecond)
		if err := openBrowser(webURL); err != nil {
			log.Printf("[WARN] 无法自动打开浏览器: %v", err)
		}
	}()
	
	if cfg.Get().Receiver.TCPEnabled {
		log.Printf("[TCP] TCP接收器: 端口 %d", cfg.Get().Receiver.TCPPort)
	}
	if cfg.Get().Receiver.UDPEnabled {
		log.Printf("[UDP] UDP接收器: 端口 %d", cfg.Get().Receiver.UDPPort)
	}
	if cfg.Get().Receiver.HTTPEnabled {
		log.Printf("[HTTP] HTTP接收器: 端口 %d", cfg.Get().Receiver.HTTPPort)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	if runtime.GOOS == "windows" {
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	} else {
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	}

	// 启动控制台输入监听 (Windows 备用方案)
	inputDone := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "exit" || text == "quit" {
				inputDone <- true
				return
			}
		}
	}()

	// 等待退出信号
	select {
	case <-sigChan:
		log.Println("[STOP] 收到终止信号，正在关闭系统...")
	case <-inputDone:
		log.Println("[STOP] 收到退出命令，正在关闭系统...")
	case err := <-serverErr:
		log.Printf("服务器错误: %v", err)
	}

	// 优雅关闭
	shutdown(ctx, srv, recvManager, proc, store)
}

// shutdown 优雅关闭系统
func shutdown(ctx context.Context, srv *server.Server, recv *receiver.Manager, proc *processor.Processor, store storage.Storage) {
	// 使用带超时的上下文
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		log.Println("[...] 正在停止接收器...")
		if err := recv.Stop(); err != nil {
			log.Printf("停止接收器出错: %v", err)
		}
		log.Println("[OK] 接收器已停止")

		log.Println("[...] 正在停止处理器...")
		proc.Stop()
		log.Println("[OK] 处理器已停止")

		log.Println("[...] 正在关闭存储...")
		if err := store.Close(); err != nil {
			log.Printf("关闭存储出错: %v", err)
		}
		log.Println("[OK] 存储已关闭")
		
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("[OK] 系统已安全关闭")
	case <-shutdownCtx.Done():
		log.Println("[WARN] 关闭超时，强制退出")
	}
	
	// 忽略 srv 未使用
	_ = srv
	
	fmt.Println("按回车键退出...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// openBrowser 使用系统默认浏览器打开URL
func openBrowser(url string) error {
	var cmd string
	var args []string
	
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux and others
		cmd = "xdg-open"
		args = []string{url}
	}
	
	return exec.Command(cmd, args...).Start()
}
