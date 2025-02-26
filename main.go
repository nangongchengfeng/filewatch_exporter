package main

import (
	"filewatch_exporter/collector"
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

/**
 * @Author: 南宫乘风
 * @Description:
 * @File:  main.go
 * @Email: 1794748404@qq.com
 * @Date: 2025-02-26 16:46
 */
var (
	// Set during go build
	// version   string
	// gitCommit string

	// 命令行参数
	listenAddr       = flag.String("c", "8080", "An port to listen on for web interface and telemetry.")
	metricsPath      = flag.String("web.telemetry-path", "/metrics", "A path under which to expose metrics.")
	metricsNamespace = flag.String("metric.namespace", "app", "Prometheus metrics namespace, as the prefix of metrics name")
)

func Query(w http.ResponseWriter, r *http.Request) {
	//模拟业务逻辑
	//模拟业务查询耗时0~1s
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	_, _ = io.WriteString(w, "some results")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 创建一个文件监控器
	apiRequestCounter := collector.NewAPIRequestCounter(*metricsNamespace)
	registry := prometheus.NewRegistry()
	registry.MustRegister(apiRequestCounter)
	// 设置HTTP服务器以处理Prometheus指标的HTTP请求
	http.Handle(*metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// 设置根路径的处理函数，用于返回一个简单的HTML页面，包含指向指标页面的链接
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	            <head><title>A Prometheus Exporter</title></head>
	            <body>
	            <h1>A Prometheus Exporter</h1>
	            <p><a href='/metrics'>Metrics</a></p>
	            </body>
	            </html>`))
	})
	// 模拟API请求的处理函数
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiRequestCounter.IncrementRequestCount()
		// 模拟API处理时间
		w.Write([]byte("API请求处理成功"))
	})

	// 记录启动日志并启动HTTP服务器监听
	log.Printf("Starting Server at http://localhost:%s%s", *listenAddr, *metricsPath)
	log.Fatal(http.ListenAndServe(":"+*listenAddr, nil))
}
