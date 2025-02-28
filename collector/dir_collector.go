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
	config    *config.Config
	dirSize   *prometheus.Desc
	dirExists *prometheus.Desc
	dirCount  *prometheus.Desc
	mutex     sync.Mutex
	sizes     map[string]float64
	exists    map[string]float64
	counts    map[string]float64
	lastScan  time.Time
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
		dirExists: prometheus.NewDesc(
			"filewatch_dir_exists",
			"Indicates whether a directory exists (1) or not (0)",
			[]string{"path"},
			nil,
		),
		dirCount: prometheus.NewDesc(
			"filewatch_dir_count",
			"Total number of files in directory",
			[]string{"path"},
			nil,
		),
		sizes:    make(map[string]float64),
		exists:   make(map[string]float64),
		counts:   make(map[string]float64),
		lastScan: time.Now(),
	}

	// 启动后台监控goroutine
	go collector.monitor()

	return collector
}

// Describe 实现Collector接口
func (c *DirCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dirSize
	ch <- c.dirExists
	ch <- c.dirCount
}

// Collect 实现Collector接口
func (c *DirCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for path, exists := range c.exists {
		ch <- prometheus.MustNewConstMetric(
			c.dirExists,
			prometheus.GaugeValue,
			exists,
			path,
		)
		// 只有当目录存在时才输出大小和文件数量指标
		if exists == 1 {
			ch <- prometheus.MustNewConstMetric(
				c.dirSize,
				prometheus.GaugeValue,
				c.sizes[path],
				path,
			)
			ch <- prometheus.MustNewConstMetric(
				c.dirCount,
				prometheus.GaugeValue,
				c.counts[path],
				path,
			)
		}
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

		// 检查目录是否存在
		exists := c.checkDirExists(dirPath)
		if oldExists, ok := c.exists[dirPath]; !ok || oldExists != exists {
			log.Printf("Directory existence change: %s exists = %.0f", dirPath, exists)
		}
		c.exists[dirPath] = exists

		// 只有当目录存在时才计算大小和文件数量
		if exists == 1 {
			// 计算目录大小
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

			// 计算文件数量
			count, err := c.calculateFileCount(dirPath)
			if err != nil {
				log.Printf("Error calculating file count for directory %s: %v", dirPath, err)
				continue
			}

			// 检查文件数量是否发生变化
			if oldCount, ok := c.counts[dirPath]; !ok || oldCount != count {
				log.Printf("Directory file count change: %s count = %.0f files", dirPath, count)
			}
			c.counts[dirPath] = count
		}
	}
}

// checkDirExists 检查目录是否存在
func (c *DirCollector) checkDirExists(path string) float64 {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0 // 目录不存在
		}
		log.Printf("Error checking directory %s: %v", path, err)
		return 0 // 发生错误也视为不存在
	}
	if !info.IsDir() {
		return 0 // 路径存在但不是目录
	}
	return 1 // 目录存在
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

// calculateFileCount 计算目录中的文件总数
func (c *DirCollector) calculateFileCount(path string) (float64, error) {
	var count int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// 如果遇到权限错误或其他错误，记录日志但继续处理
			log.Printf("Warning: error accessing path %s: %v", path, err)
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return float64(count), nil
}
