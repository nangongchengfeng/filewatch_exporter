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
	// 输出看看是否正确读取参数
	fmt.Println("使用的配置文件:", *configFile)

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
	//prometheus.MustRegister(fileCollector)

	// 创建并注册目录监控收集器
	dirCollector := collector.NewDirCollector(&cfg)
	//prometheus.MustRegister(dirCollector)

	// 创建自定义注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(fileCollector)
	registry.MustRegister(dirCollector)
	// 启动HTTP服务器
	//http.Handle(cfg.Server.MetricsPath, promhttp.Handler())
	// 启动HTTP服务器，仅暴露自定义注册表中的指标
	http.Handle(cfg.Server.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>File Monitor Exporter</title></head>
			<body>
			<h1>File Monitor Exporter</h1>
			<p>Monitoring files:</p>
			<ul>
				` + getMonitoredFilesHTML(cfg.Files) + `
			</ul>
			<p>Monitoring directories:</p>
			<ul>
				` + getMonitoredDirsHTML(cfg.Dirs) + `
			</ul>
			<p><a href="` + cfg.Server.MetricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Printf("Starting File Monitor Exporter on %s", cfg.Server.ListenAddress)
	log.Printf("Monitoring %d files: %s", len(cfg.Files), strings.Join(cfg.Files, ", "))
	log.Printf("Monitoring %d directories: %s", len(cfg.Dirs), strings.Join(cfg.Dirs, ", "))
	log.Fatal(http.ListenAndServe(cfg.Server.ListenAddress, nil))
}

// 生成监控文件列表的HTML
func getMonitoredFilesHTML(files []string) string {
	var html strings.Builder
	for _, file := range files {
		html.WriteString("<li>" + file + "</li>")
	}
	return html.String()
}

// 生成监控目录列表的HTML
func getMonitoredDirsHTML(dirs []string) string {
	var html strings.Builder
	for _, dir := range dirs {
		html.WriteString("<li>" + dir + "</li>")
	}
	return html.String()
}
