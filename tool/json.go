package tool

import (
	"encoding/json"
	"fmt"
	"github.com/kaptinlin/jsonrepair"
	"strings"
)

// GetJsonText JSON 提取函数，从字符串中提取第一个有效的 JSON
func GetJsonText(text string) (string, error) {
	text = strings.TrimSpace(text)

	// 方法1：尝试完整字符串验证
	if json.Valid([]byte(text)) {
		return text, nil
	}

	// 方法2：使用括号匹配算法提取
	extracted, err := extractJSONByBracketMatching(text)
	if err == nil && json.Valid([]byte(extracted)) {
		return extracted, nil
	}

	// 方法3：尝试修复
	result, err := jsonrepair.JSONRepair(text)
	if err != nil {
		return "", err
	}

	if !json.Valid([]byte(result)) {
		return "", fmt.Errorf("修复后的 JSON 仍然无效")
	}

	return result, nil
}

// extractJSONByBracketMatching 使用括号匹配算法提取 JSON
func extractJSONByBracketMatching(text string) (string, error) {
	// 找到第一个有效的起始字符
	start := -1
	startChar := byte(0)

	for i := 0; i < len(text); i++ {
		if text[i] == '{' || text[i] == '[' {
			start = i
			startChar = text[i]
			break
		}
	}

	if start == -1 {
		return "", fmt.Errorf("未找到 JSON 起始字符")
	}

	// 确定对应的结束字符
	endChar := byte('}')
	if startChar == '[' {
		endChar = ']'
	}

	// 使用栈进行括号匹配
	stack := 0
	inString := false
	escapeNext := false

	for i := start; i < len(text); i++ {
		ch := text[i]

		// 处理转义字符
		if escapeNext {
			escapeNext = false
			continue
		}

		if ch == '\\' {
			escapeNext = true
			continue
		}

		// 处理字符串边界
		if ch == '"' && !escapeNext {
			inString = !inString
			continue
		}

		// 如果在字符串内，跳过括号处理
		if inString {
			continue
		}

		// 处理括号
		if ch == startChar {
			stack++
		} else if ch == endChar {
			stack--
			if stack == 0 {
				// 找到匹配的结束位置
				return text[start : i+1], nil
			}
		}
	}

	// 没有找到完整匹配，返回部分结果
	return text[start:], fmt.Errorf("未找到完整的 JSON 匹配")
}
