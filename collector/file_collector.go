package collector

import (
	"crypto/md5"
	"filewatch_exporter/config"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	config      *config.Config     // 配置
	fileExists  *prometheus.Desc   // 文件是否存在
	fileChanges *prometheus.Desc   // 文件变化次数
	fileChmod   *prometheus.Desc   // 文件修改权限
	fileSize    *prometheus.Desc   // 文件大小
	mutex       sync.Mutex         // 互斥锁
	states      map[string]float64 // 文件状态
	changes     map[string]float64 // 文件变化次数
	hashes      map[string]string  // 文件哈希
	permissions map[string]float64 // 文件权限
	sizes       map[string]float64 // 文件大小
	lastReset   time.Time          // 上次重置时间
	// 存储展开后的文件列表
	expandedFiles map[string]bool // 展开后的文件列表
}

// NewFileCollector 函数用于创建一个新的FileCollector实例
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
		fileChmod: prometheus.NewDesc(
			"filewatch_file_chmod",
			"Current file permissions in numeric format (e.g. 644)",
			[]string{"path"},
			nil,
		),
		fileSize: prometheus.NewDesc(
			"filewatch_file_size",
			"Current file size in bytes",
			[]string{"path"},
			nil,
		),
		states:        make(map[string]float64),
		changes:       make(map[string]float64),
		hashes:        make(map[string]string),
		permissions:   make(map[string]float64),
		sizes:         make(map[string]float64),
		lastReset:     time.Now(),
		expandedFiles: make(map[string]bool),
	}

	// 初始展开文件通配符
	collector.expandGlobPatterns()

	go collector.monitor()
	go collector.resetCounter()

	return collector
}

// Describe 实现Collector接口
func (c *FileCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fileExists
	ch <- c.fileChanges
	ch <- c.fileChmod
	ch <- c.fileSize
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
		if exists == 1 {
			ch <- prometheus.MustNewConstMetric(
				c.fileChmod,
				prometheus.GaugeValue,
				c.permissions[path],
				path,
			)
			ch <- prometheus.MustNewConstMetric(
				c.fileSize,
				prometheus.GaugeValue,
				c.sizes[path],
				path,
			)
		}
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

// expandGlobPatterns 展开所有的通配符模式为实际的文件路径
func (c *FileCollector) expandGlobPatterns() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	newExpandedFiles := make(map[string]bool)

	for _, pattern := range c.config.Files {
		// 检查是否包含通配符
		if containsGlobChar(pattern) {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				log.Printf("Error expanding glob pattern %s: %v", pattern, err)
				continue
			}
			// 添加匹配到的文件
			for _, match := range matches {
				newExpandedFiles[match] = true
				if _, exists := c.expandedFiles[match]; !exists {
					log.Printf("New file matched by pattern %s: %s", pattern, match)
				}
			}
		} else {
			// 不包含通配符的直接添加
			newExpandedFiles[pattern] = true
		}
	}

	// 检查移除的文件
	for oldFile := range c.expandedFiles {
		if !newExpandedFiles[oldFile] {
			log.Printf("File no longer matched by any pattern: %s", oldFile)
			// 清理相关状态
			delete(c.states, oldFile)
			delete(c.changes, oldFile)
			delete(c.hashes, oldFile)
			delete(c.permissions, oldFile)
			delete(c.sizes, oldFile)
		}
	}

	c.expandedFiles = newExpandedFiles
}

// containsGlobChar 检查路径是否包含通配符
func containsGlobChar(path string) bool {
	return strings.ContainsAny(path, "*?[]")
}

// checkFiles 检查所有配置的文件
func (c *FileCollector) checkFiles() {
	// 首先展开通配符
	c.expandGlobPatterns()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 遍历展开后的文件列表
	for path := range c.expandedFiles {
		exists := c.checkFileExists(path)

		// Update existence state
		if oldState, ok := c.states[path]; !ok || oldState != exists {
			log.Printf("File status change: %s exists = %.0f", path, exists)
		}
		c.states[path] = exists

		// Only check permissions, size and content if file exists
		if exists == 1 {
			// Check file permissions
			currentPerms := c.getFilePermissions(path)
			if oldPerms, ok := c.permissions[path]; !ok || oldPerms != currentPerms {
				log.Printf("File permissions change: %s permissions = %.0f", path, currentPerms)
			}
			c.permissions[path] = currentPerms

			// Check file size
			currentSize := c.getFileSize(path)
			if oldSize, ok := c.sizes[path]; !ok || oldSize != currentSize {
				log.Printf("File size change: %s size = %.0f bytes", path, currentSize)
			}
			c.sizes[path] = currentSize

			// Check content changes
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

// getFilePermissions returns the file permissions in numeric format (e.g. 644)
func (c *FileCollector) getFilePermissions(path string) float64 {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("Error getting permissions for %s: %v", path, err)
		return 0
	}

	// Convert os.FileMode to octal format (e.g. 0600 -> 600)
	mode := info.Mode().Perm()
	// Calculate octal value manually
	octal := float64(((mode>>6)&7)*100 + ((mode>>3)&7)*10 + (mode & 7))
	return octal
}

// getFileSize returns the file size in bytes
func (c *FileCollector) getFileSize(path string) float64 {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("Error getting size for %s: %v", path, err)
		return 0
	}
	return float64(info.Size())
}
