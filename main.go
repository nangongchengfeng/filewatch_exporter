package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

/**
 * @Author: 南宫乘风
 * @Description:
 * @File:  main.go
 * @Email: 1794748404@qq.com
 * @Date: 2025-02-26 16:46
 */

// Config 结构体定义了配置文件的结构
type Config struct {
	Server struct {
		ListenAddress string `yaml:"listen_address"`
		MetricsPath   string `yaml:"metrics_path"`
	} `yaml:"server"`
	Files    []string `yaml:"files"`
	Interval int      `yaml:"check_interval_seconds"`
}

var (
	// 定义配置文件路径
	//configFile = flag.String("conf", "config.yaml", "Path to configuration file")
	configFile = "conf/config.yaml"
	// 定义一个gauge类型的指标，用于表示文件是否存在
	fileExistsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "filewatch_file_exists",
			Help: "Indicates whether a file exists (1) or not (0)",
		},
		[]string{"path"},
	)

	// 全局配置对象
	config Config
)

// 初始化函数，注册指标
func init() {
	// 注册指标到Prometheus默认注册表
	prometheus.MustRegister(fileExistsMetric)
}

// 加载YAML配置文件
func loadConfig(configPath string) (Config, error) {
	var config Config

	// 设置默认值
	config.Server.ListenAddress = ":9100"
	config.Server.MetricsPath = "/metrics"
	config.Interval = 10

	// 读取配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	// 解析YAML
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// 检查文件是否存在
func checkFileExists(path string) float64 {
	_, err := os.Stat(path)
	if err == nil {
		return 1 // 文件存在
	}
	if os.IsNotExist(err) {
		return 0 // 文件不存在
	}
	log.Printf("Error checking file %s: %v", path, err)
	return 0 // 发生错误也视为不存在
}

// 更新所有监控文件的状态指标
func updateFileMetrics() {
	for {
		for _, filePath := range config.Files {
			exists := checkFileExists(filePath)
			fileExistsMetric.WithLabelValues(filePath).Set(exists)
			log.Printf("Updated metrics for %s: exists = %.0f", filePath, exists)
		}

		// 根据配置的间隔时间检查文件状态
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func main() {
	flag.Parse()

	// 加载配置
	var err error
	config, err = loadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// 启动一个goroutine定期更新文件状态指标
	go updateFileMetrics()

	// 配置HTTP服务器来暴露指标
	http.Handle(config.Server.MetricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>File Monitor Exporter</title></head>
			<body>
			<h1>File Monitor Exporter</h1>
			<p>Monitoring files:</p>
			<ul>
				` + getMonitoredFilesHTML() + `
			</ul>
			<p><a href="` + config.Server.MetricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Printf("Starting File Monitor Exporter on %s", config.Server.ListenAddress)
	log.Printf("Monitoring %d files: %s", len(config.Files), strings.Join(config.Files, ", "))
	log.Fatal(http.ListenAndServe(config.Server.ListenAddress, nil))
}

// 生成监控文件列表的HTML
func getMonitoredFilesHTML() string {
	var html strings.Builder
	for _, file := range config.Files {
		html.WriteString("<li>" + file + "</li>")
	}
	return html.String()
}
