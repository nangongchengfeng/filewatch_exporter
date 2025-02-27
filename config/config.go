package config

import "io/ioutil"
import "gopkg.in/yaml.v2"

/**
 * @Author: 南宫乘风
 * @Description:
 * @File:  config.go
 * @Email: 1794748404@qq.com
 * @Date: 2025-02-26 17:43
 */

// Config 结构体定义了配置文件的结构
type Config struct {
	Server struct {
		ListenAddress string `yaml:"listen_address"`
		MetricsPath   string `yaml:"metrics_path"`
	} `yaml:"server"`
	Files    []string `yaml:"files"`
	Interval int      `yaml:"check_interval_seconds"`
	Reset    int      `yaml:"reset_interval_minutes"`
}

// LoadConfig 加载并解析YAML配置文件
func LoadConfig(configPath string) (Config, error) {
	var config Config

	// 设置默认值
	config.Server.ListenAddress = ":9100"
	config.Server.MetricsPath = "/metrics"
	config.Interval = 10
	config.Reset = 30

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
