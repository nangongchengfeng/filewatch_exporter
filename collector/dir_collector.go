package collector

import (
	"filewatch_exporter/config"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DirCollector 实现了Prometheus.Collector接口，用于监控目录大小
type DirCollector struct {
	config   *config.Config
	dirSize  *prometheus.Desc
	mutex    sync.Mutex
	sizes    map[string]float64
	lastScan time.Time
}

// NewDirCollector 创建一个新的目录收集器
func NewDirCollector(config *config.Config) *DirCollector {
	collector := &DirCollector{
		config: config,
		dirSize: prometheus.NewDesc(
			"filewatch_dir_size",
			"Total size of directory in bytes",
			[]string{"path"},
			nil,
		),
		sizes:    make(map[string]float64),
		lastScan: time.Now(),
	}

	// 启动后台监控goroutine
	go collector.monitor()

	return collector
}

// Describe 实现Collector接口
func (c *DirCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dirSize
}

// Collect 实现Collector接口
func (c *DirCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for path, size := range c.sizes {
		ch <- prometheus.MustNewConstMetric(
			c.dirSize,
			prometheus.GaugeValue,
			size,
			path,
		)
	}
}

// monitor 执行定期目录大小检查
func (c *DirCollector) monitor() {
	for {
		c.checkDirs()
		time.Sleep(time.Duration(c.config.Interval) * time.Second)
	}
}

// checkDirs 检查所有配置的目录
func (c *DirCollector) checkDirs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, dirPath := range c.config.Dirs {
		// 确保路径以斜杠结尾
		if !os.IsPathSeparator(dirPath[len(dirPath)-1]) {
			dirPath = dirPath + string(os.PathSeparator)
		}

		size, err := c.calculateDirSize(dirPath)
		if err != nil {
			log.Printf("Error calculating size for directory %s: %v", dirPath, err)
			continue
		}

		// 检查大小是否发生变化
		if oldSize, ok := c.sizes[dirPath]; !ok || oldSize != size {
			log.Printf("Directory size change: %s size = %.0f bytes", dirPath, size)
		}
		c.sizes[dirPath] = size
	}
}

// calculateDirSize 计算目录的总大小
func (c *DirCollector) calculateDirSize(path string) (float64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// 如果遇到权限错误或其他错误，记录日志但继续处理
			log.Printf("Warning: error accessing path %s: %v", path, err)
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return float64(size), nil
}
