# Filewatch Exporter

基于Go语言开发的文件系统监控工具，用于监控文件和目录变化并通过Prometheus进行指标采集的exporter。

## 项目简介

Filewatch Exporter 是一个轻量级的监控工具，专门用于监控文件系统的变化。它可以实时检测文件和目录的变动，并将这些变化转换为Prometheus可以采集的指标数据，便于系统运维和告警管理。

## 核心功能

### 文件监控
- 监控文件修改时间变化
- 监控文件大小变化
- 监控文件内容变化（支持MD5校验）
- 监控文件权限变更
- 支持文件存在性检查

### 目录监控
- 监控目录下文件数量变化
- 监控目录总大小变化
- 支持递归监控子目录
- 支持文件模式匹配（通配符）
- 监控目录权限变更

### 告警功能
- 支持自定义告警规则
- 支持多种告警阈值设置
- 支持告警延迟和冷却时间
- 与Prometheus AlertManager集成

### 配置管理
- 支持YAML格式配置文件
- 支持热重载配置
- 支持多个监控目标配置
- 支持监控项的自定义标签

## 技术选型

### 基础框架
- 开发语言: Go 1.20+
- 依赖管理: Go Modules

### 核心依赖
- `github.com/prometheus/client_golang`: Prometheus客户端库
- `github.com/fsnotify/fsnotify`: 文件系统事件通知
- `gopkg.in/yaml.v3`: YAML配置解析
- `github.com/sirupsen/logrus`: 结构化日志
- `github.com/gin-gonic/gin`: Web框架

## 监控指标

### 文件指标
- `filewatch_file_exists{path="/path/to/file"} 1|0`
- `filewatch_file_size_bytes{path="/path/to/file"} 1234`
- `filewatch_file_modified_time_seconds{path="/path/to/file"} 1234567890`
- `filewatch_file_permission{path="/path/to/file"} 644`

### 目录指标
- `filewatch_directory_files_total{path="/path/to/dir"} 10`
- `filewatch_directory_size_bytes{path="/path/to/dir"} 1234567`
- `filewatch_directory_changes_total{path="/path/to/dir",type="create|modify|delete"} 5`

### 系统指标
- `filewatch_scrape_duration_seconds{} 0.1`
- `filewatch_scrape_errors_total{} 0`

## 配置示例

```yaml
global:
  scrape_interval: 30s
  metrics_path: /metrics
  listen_address: ":9090"

targets:
  - name: "nginx-config"
    paths:
      - "/etc/nginx/nginx.conf"
      - "/etc/nginx/conf.d/*.conf"
    recursive: false
    labels:
      service: "nginx"
      env: "production"
    checks:
      - type: "modification"
        interval: "1m"
      - type: "existence"
        interval: "30s"

  - name: "app-logs"
    paths:
      - "/var/log/app/"
    recursive: true
    labels:
      service: "application"
      env: "production"
    checks:
      - type: "size"
        interval: "5m"
      - type: "count"
        interval: "1m"
```

## 快速开始

1. 安装
```bash
go get github.com/yourusername/filewatch_exporter
```

2. 创建配置文件
```bash
cp config.example.yml config.yml
vim config.yml
```

3. 运行exporter
```bash
filewatch_exporter --config.file=config.yml
```

4. 访问metrics接口
```bash
curl http://localhost:9090/metrics
```

## Prometheus配置

```yaml
scrape_configs:
  - job_name: 'filewatch'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

## 构建和部署

### 本地构建
```bash
make build
```

### Docker构建
```bash
docker build -t filewatch_exporter .
docker run -d -p 9090:9090 -v /path/to/config.yml:/etc/filewatch/config.yml filewatch_exporter
```

## 贡献指南

欢迎提交Issue和Pull Request来帮助改进项目。在提交PR之前，请确保：

1. 代码已经通过测试
2. 新功能已添加测试用例
3. 文档已更新

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。
