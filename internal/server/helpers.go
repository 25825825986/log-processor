// helpers.go - 辅助函数
package server

import (
	"encoding/json"
	"fmt"
	"log-processor/internal/config"
	"strings"
)

// 辅助函数

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// mergeConfigSection 合并配置节
func mergeConfigSection(payload map[string]interface{}, key string, target interface{}) error {
	if section, ok := payload[key]; ok {
		data, err := json.Marshal(section)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", key, err)
		}
		if err := json.Unmarshal(data, target); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", key, err)
		}
	}
	return nil
}

// compareReceiverConfig 比较两个接收器配置是否相同
func compareReceiverConfig(a, b config.ReceiverConfig) bool {
	return a.TCPEnabled == b.TCPEnabled &&
		a.TCPPort == b.TCPPort &&
		a.UDPEnabled == b.UDPEnabled &&
		a.UDPPort == b.UDPPort &&
		a.HTTPEnabled == b.HTTPEnabled &&
		a.HTTPPort == b.HTTPPort &&
		a.BufferSize == b.BufferSize &&
		a.MaxConnections == b.MaxConnections &&
		a.HTTPAuthToken == b.HTTPAuthToken &&
		a.HTTPRateLimit == b.HTTPRateLimit &&
		a.HTTPMaxBodySize == b.HTTPMaxBodySize &&
		compareStringSlices(a.HTTPAllowedIPs, b.HTTPAllowedIPs) &&
		a.FileWatcherEnabled == b.FileWatcherEnabled &&
		compareStringSlices(a.WatchPaths, b.WatchPaths)
}

// compareStorageConfig 比较两个存储配置是否相同
func compareStorageConfig(a, b config.StorageConfig) bool {
	return a.Type == b.Type &&
		a.DBPath == b.DBPath &&
		a.RetentionHours == b.RetentionHours &&
		a.MaxMemoryItems == b.MaxMemoryItems
}

// compareStringSlices 比较两个字符串切片是否相同
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// getExtension 根据格式返回文件扩展名
func getExtension(format string) string {
	switch strings.ToLower(format) {
	case "excel":
		return ".xlsx"
	case "csv":
		return ".csv"
	case "json":
		return ".json"
	default:
		return ".txt"
	}
}

// asInt64 将 interface{} 转换为 int64
func asInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// boolToFloat64 将布尔值转换为 float64（用于 metrics）
func boolToFloat64(v bool) float64 {
	if v {
		return 1.0
	}
	return 0.0
}

// writePromMetric 写入 Prometheus 格式指标
func writePromMetric(builder *strings.Builder, name string, value float64) {
	builder.WriteString(fmt.Sprintf("%s %.2f\n", name, value))
}

// metricDelta 计算两个 map 中某个 key 的差值
func metricDelta(after map[string]interface{}, before map[string]interface{}, key string) int64 {
	return asInt64(after[key]) - asInt64(before[key])
}
