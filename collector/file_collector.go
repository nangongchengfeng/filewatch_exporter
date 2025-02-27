package collector

import (
	"crypto/md5"
	"filewatch_exporter/config"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

/**
 * @Author: 南宫乘风
 * @Description:
 * @File:  file_collector.go
 * @Email: 1794748404@qq.com
 * @Date: 2025-02-27 11:30
 */

// FileCollector 实现了Prometheus.Collector接口
type FileCollector struct {
	config      *config.Config
	fileExists  *prometheus.Desc
	fileChanges *prometheus.Desc
	mutex       sync.Mutex
	states      map[string]float64
	changes     map[string]float64
	hashes      map[string]string
	lastReset   time.Time
}

// NewFileCollector 创建一个新的文件收集器
func NewFileCollector(config *config.Config) *FileCollector {
	collector := &FileCollector{
		config: config,
		fileExists: prometheus.NewDesc(
			"filewatch_file_exists",
			"Indicates whether a file exists (1) or not (0)",
			[]string{"path"},
			nil,
		),
		fileChanges: prometheus.NewDesc(
			"filewatch_file_change",
			"Number of times the file has changed since last reset",
			[]string{"path"},
			nil,
		),
		states:    make(map[string]float64),
		changes:   make(map[string]float64),
		hashes:    make(map[string]string),
		lastReset: time.Now(),
	}

	// 启动后台监控goroutine
	go collector.monitor()
	go collector.resetCounter() // Start the reset counter goroutine

	return collector
}

// Describe 实现Collector接口
func (c *FileCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fileExists
	ch <- c.fileChanges
}

// Collect 实现Collector接口
func (c *FileCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for path, exists := range c.states {
		ch <- prometheus.MustNewConstMetric(
			c.fileExists,
			prometheus.GaugeValue,
			exists,
			path,
		)
		ch <- prometheus.MustNewConstMetric(
			c.fileChanges,
			prometheus.GaugeValue,
			c.changes[path],
			path,
		)
	}
}

// monitor 执行定期文件检查
func (c *FileCollector) monitor() {
	for {
		c.checkFiles()
		time.Sleep(time.Duration(c.config.Interval) * time.Second)
	}
}

// resetCounter 执行定期重置文件变化计数器
func (c *FileCollector) resetCounter() {
	for {
		time.Sleep(time.Duration(c.config.Reset) * time.Minute)
		c.mutex.Lock()
		log.Println("Resetting file change counters")
		for path := range c.changes {
			c.changes[path] = 0
		}
		c.lastReset = time.Now()
		c.mutex.Unlock()
	}
}

// checkFiles 检查所有配置的文件
func (c *FileCollector) checkFiles() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, path := range c.config.Files {
		exists := c.checkFileExists(path)

		// Update existence state
		if oldState, ok := c.states[path]; !ok || oldState != exists {
			log.Printf("File status change: %s exists = %.0f", path, exists)
		}
		c.states[path] = exists

		// Check content changes only if file exists
		if exists == 1 {
			currentHash, err := c.calculateFileHash(path)
			if err != nil {
				log.Printf("Error calculating hash for %s: %v", path, err)
				continue
			}

			// Initialize change counter if not exists
			if _, ok := c.changes[path]; !ok {
				c.changes[path] = 0
			}

			// Compare with previous hash
			if previousHash, ok := c.hashes[path]; ok && previousHash != currentHash {
				c.changes[path]++
				log.Printf("File content change detected for %s, change count: %.0f", path, c.changes[path])
			}
			c.hashes[path] = currentHash
		}
	}
}

// calculateFileHash 计算文件的MD5哈希值
func (c *FileCollector) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return string(hash.Sum(nil)), nil
}

// checkFileExists 检查单个文件是否存在
func (c *FileCollector) checkFileExists(path string) float64 {
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
