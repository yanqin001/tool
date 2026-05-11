package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaptinlin/jsonrepair"
)

// GetJsonText JSON 提取函数，从字符串中提取第一个有效的 JSON
func GetJsonText(text string) (string, error) {
	return getJsonTextWithDepth(text, 0)
}

// getJsonTextWithDepth 带递归深度保护的 JSON 提取实现，防止多层 JSON-encode 导致无限递归
func getJsonTextWithDepth(text string, depth int) (string, error) {
	text = strings.TrimSpace(text)

	// 方法0：若整体被双引号包裹且是合法的 JSON 字符串值（常见于 LLM 响应被二次 JSON-encode 的场景），
	// 先解码得到原始文本再递归提取；限制递归深度避免极端输入造成栈溢出
	if depth < 3 && len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		var inner string
		if err := json.Unmarshal([]byte(text), &inner); err == nil && strings.ContainsAny(inner, "{[") {
			if result, subErr := getJsonTextWithDepth(inner, depth+1); subErr == nil {
				return result, nil
			}
		}
	}

	// 方法1：尝试完整字符串验证（仅当顶层是对象或数组时才直接返回，
	// 避免把“被引号包裹的字符串”误判为目标 JSON）
	if json.Valid([]byte(text)) && len(text) > 0 && (text[0] == '{' || text[0] == '[') {
		return text, nil
	}

	// 方法2：使用括号匹配算法提取
	extracted, err := extractJSONByBracketMatching(text)
	if err == nil && json.Valid([]byte(extracted)) {
		return extracted, nil
	}

	// 方法3：对提取出的片段优先尝试修复（适用于 JSON 被截断或轻微损坏的场景）
	if extracted != "" {
		if repaired, repairErr := jsonrepair.JSONRepair(extracted); repairErr == nil && json.Valid([]byte(repaired)) {
			return repaired, nil
		}
	}

	// 方法4：兜底，尝试修复整段文本
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

	// 没有找到完整匹配，根据栈深度补齐结束字符，返回尽力匹配的结果交由上层修复
	partial := text[start:]
	if stack > 0 {
		// 如果当前仍处于字符串中，先补一个引号闭合字符串
		if inString {
			partial += "\""
		}
		// 按栈深度补齐剩余的结束符
		for i := 0; i < stack; i++ {
			partial += string(endChar)
		}
	}
	return partial, fmt.Errorf("未找到完整的 JSON 匹配")
}
