// parser/auto_detect.go - Auto format detection
package parser

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	reNginxDetect    = regexp.MustCompile(`^\S+\s+\S+\s+\S+\s+\[[^\]]+\]\s+"[^"]+"\s+\d+\s+\d+`)
	reNginxTimeSuffix = regexp.MustCompile(`"\d+\.\d+"$`)
	reSyslogDetect   = regexp.MustCompile(`^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d+\s+\d+:\d+:\d+`)
	reNginxLog       = regexp.MustCompile(`^(?P<client_ip>\S+)\s+\S+\s+\S+\s+\[(?P<timestamp>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<protocol>[^"]+)"\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)(?:\s+"(?P<referer>[^"]*)"\s+"(?P<user_agent>[^"]*)"(?:\s+"(?P<response_time>[^"]*)")?|\s+(?P<response_time>[\d.]+))?`)
)

// 算法2-4：日志格式识别算法
// DetectFormat automatically detects log format
func DetectFormat(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 {
		return "unknown"
	}

	// 1. Detect JSON format
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var js map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &js); err == nil {
			return "json"
		}
	}

	// 2. Detect Nginx/Apache format (Combined Log Format)
	if reNginxDetect.MatchString(trimmed) {
		if strings.HasSuffix(trimmed, `"`) || reNginxTimeSuffix.MatchString(trimmed) {
			return "nginx"
		}
		return "apache"
	}

	// 3. Detect CSV format
	if strings.Count(trimmed, ",") >= 3 && !strings.Contains(trimmed, `"`) {
		fields := strings.Split(trimmed, ",")
		if len(fields) >= 4 {
			return "csv"
		}
	}

	// 4. Detect TSV format (tab-separated)
	if strings.Count(trimmed, "\t") >= 3 {
		return "tsv"
	}

	// 5. Detect Syslog format
	if reSyslogDetect.MatchString(trimmed) {
		return "syslog"
	}

	// 6. Detect common delimiter formats (pipe, semicolon, etc.)
	if strings.Count(trimmed, "|") >= 3 {
		return "pipe"
	}
	if strings.Count(trimmed, ";") >= 3 {
		return "semicolon"
	}

	// 8. Plain text/unstructured
	return "plain"
}

// parseNginxLog parses Nginx/Apache format log
func parseNginxLog(line string) (map[string]string, bool) {
	result := make(map[string]string)
	
	matches := reNginxLog.FindStringSubmatch(line)
	if matches == nil {
		return result, false
	}
	
	names := reNginxLog.SubexpNames()
	for i, name := range names {
		if i > 0 && i < len(matches) && name != "" {
			if _, exists := result[name]; !exists || matches[i] != "" {
				result[name] = matches[i]
			}
		}
	}
	
	return result, true
}
