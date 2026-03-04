// main.go - 程序入口
package main

import (
	"flag"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/parser"
	"log-processor/internal/processor"
	"log-processor/internal/receiver"
	"log-processor/internal/server"
	"log-processor/internal/storage"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./config.json", "配置文件路径")
	flag.Parse()

	log.Println("🚀 启动日志数据处理系统...")

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
	store, err := storage.NewSQLiteStorage(cfg.Get().Storage)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}
	defer store.Close()

	// 初始化解析器
	parserCfg := cfg.GetParserConfig()
	logParser := parser.NewLogParser(parserCfg)

	// 初始化处理器
	processorCfg := cfg.GetProcessorConfig()
	proc := processor.NewProcessor(processorCfg, logParser, store)
	proc.Start()
	defer proc.Stop()

	// 初始化接收器
	receiverCfg := cfg.GetReceiverConfig()
	recvManager := receiver.NewManager(receiverCfg)

	// 启动接收器
	err = recvManager.Start(func(line string) {
		// 提交到处理器
		if !proc.Submit(line) {
			log.Printf("处理器队列已满，丢弃日志: %s", line[:min(50, len(line))])
		}
	})
	if err != nil {
		log.Fatalf("启动接收器失败: %v", err)
	}
	defer recvManager.Stop()

	// 初始化Web服务器
	srv := server.NewServer(cfg, store, proc, recvManager)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在后台启动服务器
	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("启动Web服务器失败: %v", err)
		}
	}()

	log.Println("✅ 系统启动成功！")
	log.Printf("📊 Web界面: http://localhost:%d", cfg.Get().Server.Port)
	if cfg.Get().Receiver.TCPEnabled {
		log.Printf("📡 TCP接收器: 端口 %d", cfg.Get().Receiver.TCPPort)
	}
	if cfg.Get().Receiver.UDPEnabled {
		log.Printf("📡 UDP接收器: 端口 %d", cfg.Get().Receiver.UDPPort)
	}
	if cfg.Get().Receiver.HTTPEnabled {
		log.Printf("📡 HTTP接收器: 端口 %d", cfg.Get().Receiver.HTTPPort)
	}

	// 等待退出信号
	<-quit
	log.Println("🛑 正在关闭系统...")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
