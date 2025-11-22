// livechat-server 项目的程序入口，负责解析启动参数并启动聊天室服务。
package main

import (
	"context"
	"flag"
	"livechat-server/internal/chat"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main 解析命令行参数、构造配置并阻塞式运行 TCP 聊天服务器。
func main() {
	var (
		addr    string
		idle    time.Duration
		history int
		maxMsg  int
	)

	// 可运行时调节的参数，方便在本地练习时快速修改行为。
	flag.StringVar(&addr, "addr", ":8080", "TCP address to bind (e.g. :8080 or 127.0.0.1:5000)")
	flag.DurationVar(&idle, "idle", 2*time.Minute, "Idle timeout before a user is disconnected")
	flag.IntVar(&history, "history", 50, "Number of messages remembered per room for /history")
	flag.IntVar(&maxMsg, "max-bytes", 4096, "Maximum number of bytes accepted per message")
	flag.Parse()

	// 基于默认配置生成当前会话的配置实体。
	cfg := chat.DefaultConfig()
	cfg.IdleTimeout = idle
	cfg.HistoryLimit = history
	cfg.MaxMessageBytes = maxMsg

	srv := chat.NewServer(cfg)
	// 使用 NotifyContext 以便 Ctrl+C 时优雅退出。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("livechat-server listening on %s", addr)
	// 阻塞直到服务器出错或收到退出信号。
	if err := srv.ListenAndServe(ctx, addr); err != nil {
		if ctx.Err() != nil {
			log.Printf("server shut down: %v", err)
		} else {
			log.Fatalf("server error: %v", err)
		}
	}
}
