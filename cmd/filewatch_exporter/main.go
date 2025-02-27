package main

import (
	"filewatch_exporter/collector"
	"filewatch_exporter/config"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"strings"
)

/**
 * @Author: 南宫乘风
 * @Description:
 * @File:  main.go
 * @Email: 1794748404@qq.com
 * @Date: 2025-02-26 16:46
 */
var (
	configFile  = flag.String("config", "config/config.yaml", "Path to configuration file")
	showVersion = flag.Bool("version", false, "Print version information")
)

// 版本信息，可从构建时注入
var (
	Version   = "development"
	BuildDate = "unknown"
	Commit    = "unknown"
)

func main() {
	flag.Parse()

	// 打印版本信息并退出
	if *showVersion {
		fmt.Printf("filewatch_exporter version %s (built: %s, commit: %s)\n",
			Version, BuildDate, Commit)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// 创建收集器
	fileCollector := collector.NewFileCollector(&cfg)

	// 注册收集器
	prometheus.MustRegister(fileCollector)

	// 启动HTTP服务器
	http.Handle(cfg.Server.MetricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHomePage(w, cfg)
	})

	log.Printf("Starting Filewatch Exporter version %s (built: %s, commit: %s)",
		Version, BuildDate, Commit)
	log.Printf("Listening on %s", cfg.Server.ListenAddress)
	log.Printf("Monitoring %d files: %s", len(cfg.Files), strings.Join(cfg.Files, ", "))
	log.Fatal(http.ListenAndServe(cfg.Server.ListenAddress, nil))
}

// 渲染主页
func renderHomePage(w http.ResponseWriter, cfg config.Config) {
	var filesHTML strings.Builder
	for _, file := range cfg.Files {
		filesHTML.WriteString(fmt.Sprintf("<li>%s</li>", file))
	}

	html := fmt.Sprintf(`<html>
	<head><title>Filewatch Exporter</title></head>
	<body>
		<h1>Filewatch Exporter</h1>
		<p>Version: %s</p>
		<p>Monitoring files:</p>
		<ul>%s</ul>
		<p><a href="%s">Metrics</a></p>
	</body>
	</html>`, Version, filesHTML.String(), cfg.Server.MetricsPath)

	w.Write([]byte(html))
}
