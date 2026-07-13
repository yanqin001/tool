package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaptinlin/jsonrepair"
)

// GetJsonText 从字符串中提取干净 JSON 文本，并优先返回第一个有效的 JSON 对象或数组。
// 参数 text 是可能包含 Markdown、说明文本或二次 JSON-encode 内容的原始响应。
// 返回值为可被 encoding/json 解析的 JSON 文本；若不存在对象或数组，会保留 jsonrepair 的兜底修复行为。
// 当无法提取或修复 JSON 时返回错误。
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

	// 方法2：优先处理 ```json 代码块。LLM 经常在正文里先给代码片段，
	// 再给最终 JSON；明确标记为 json 的代码块通常才是目标结果。
	for _, block := range extractJSONCodeBlocks(text) {
		if result, ok := rootJSONOrRepair(block); ok {
			return result, nil
		}
		if result, ok := firstValidJSONCandidate(block); ok {
			return result, nil
		}
		if result, ok := firstRepairedJSONCandidate(block); ok {
			return result, nil
		}
	}

	// 方法3：优先修复第一个“像 JSON”的外层片段，避免截断外层对象中的内部对象被误当成最终结果。
	if result, ok := firstRepairedJSONCandidate(text); ok {
		return result, nil
	}

	// 方法4：扫描所有括号配对完整的候选，而不是只取第一个 "{" 或 "["。
	// 这样可以跳过 C/Java/Python 代码块里的伪 JSON 片段，继续寻找后续真正结果。
	if result, ok := firstValidJSONCandidate(text); ok {
		return result, nil
	}

	// 方法5：兜底，尝试修复整段文本
	result, err := jsonrepair.JSONRepair(text)
	if err != nil {
		return "", err
	}

	if !json.Valid([]byte(result)) {
		return "", fmt.Errorf("修复后的 JSON 仍然无效")
	}

	return result, nil
}

// rootJSONOrRepair 仅在文本自身以 JSON 起始字符开头时工作，用于优先保留外层 JSON。
func rootJSONOrRepair(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !isJSONStart(trimmed[0]) {
		return "", false
	}
	if isValidJSONObjectOrArray(trimmed) {
		return trimmed, true
	}
	return repairJSONObjectOrArray(completePartialJSON(trimmed))
}

// extractJSONCodeBlocks 提取 Markdown 中语言标记为 json 的代码块。
func extractJSONCodeBlocks(text string) []string {
	var blocks []string
	searchFrom := 0
	for {
		startRel := strings.Index(text[searchFrom:], "```")
		if startRel < 0 {
			return blocks
		}

		fenceStart := searchFrom + startRel
		infoStart := fenceStart + len("```")
		lineEndRel := strings.IndexByte(text[infoStart:], '\n')
		if lineEndRel < 0 {
			return blocks
		}

		info := strings.TrimSpace(text[infoStart : infoStart+lineEndRel])
		bodyStart := infoStart + lineEndRel + 1
		endRel := strings.Index(text[bodyStart:], "```")
		if endRel < 0 {
			if strings.EqualFold(info, "json") {
				blocks = append(blocks, strings.TrimSpace(text[bodyStart:]))
			}
			return blocks
		}

		if strings.EqualFold(info, "json") {
			blocks = append(blocks, strings.TrimSpace(text[bodyStart:bodyStart+endRel]))
		}
		searchFrom = bodyStart + endRel + len("```")
	}
}

// firstValidJSONCandidate 返回文本中第一个括号完整且语法合法的 JSON 对象或数组。
func firstValidJSONCandidate(text string) (string, bool) {
	for start := 0; start < len(text); start++ {
		if !isJSONStart(text[start]) {
			continue
		}

		candidate, ok := matchCompleteJSON(text, start)
		if ok && isValidJSONObjectOrArray(candidate) {
			return candidate, true
		}
	}
	return "", false
}

// firstRepairedJSONCandidate 从每个像 JSON 的起始字符开始尝试修复，主要用于截断响应。
func firstRepairedJSONCandidate(text string) (string, bool) {
	for start := 0; start < len(text); start++ {
		if !isJSONStart(text[start]) {
			continue
		}

		partial := completePartialJSON(text[start:])
		if !looksLikeJSONPayload(partial) {
			continue
		}
		if repaired, ok := repairJSONObjectOrArray(partial); ok {
			return repaired, true
		}
	}
	return "", false
}

// matchCompleteJSON 使用混合括号栈匹配完整 JSON，字符串里的括号不会参与配对。
func matchCompleteJSON(text string, start int) (string, bool) {
	if start < 0 || start >= len(text) || !isJSONStart(text[start]) {
		return "", false
	}

	stack := []byte{matchingEnd(text[start])}
	inString := false
	escapeNext := false

	for i := start + 1; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escapeNext {
				escapeNext = false
				continue
			}
			switch ch {
			case '\\':
				escapeNext = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, matchingEnd(ch))
		case '}', ']':
			if len(stack) == 0 || stack[len(stack)-1] != ch {
				return "", false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return text[start : i+1], true
			}
		}
	}

	return "", false
}

// completePartialJSON 对可能被截断的 JSON 片段补齐字符串和括号，交由 jsonrepair 做最终修复。
func completePartialJSON(text string) string {
	if text == "" || !isJSONStart(text[0]) {
		return text
	}

	stack := []byte{matchingEnd(text[0])}
	inString := false
	escapeNext := false

	for i := 1; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escapeNext {
				escapeNext = false
				continue
			}
			switch ch {
			case '\\':
				escapeNext = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, matchingEnd(ch))
		case '}', ']':
			if len(stack) == 0 || stack[len(stack)-1] != ch {
				return text[:i]
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return text[:i+1]
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(text)
	if inString {
		if escapeNext {
			builder.WriteByte('\\')
		}
		builder.WriteByte('"')
	}
	for i := len(stack) - 1; i >= 0; i-- {
		builder.WriteByte(stack[i])
	}
	return builder.String()
}

func repairJSONObjectOrArray(text string) (string, bool) {
	repaired, err := jsonrepair.JSONRepair(text)
	if err != nil || !isValidJSONObjectOrArray(repaired) {
		return "", false
	}
	return repaired, true
}

func isValidJSONObjectOrArray(text string) bool {
	trimmed := strings.TrimSpace(text)
	return len(trimmed) > 0 && isJSONStart(trimmed[0]) && json.Valid([]byte(trimmed))
}

// looksLikeJSONPayload 做轻量启发式过滤，避免把代码块或普通说明中的括号误修成 JSON。
func looksLikeJSONPayload(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !isJSONStart(trimmed[0]) {
		return false
	}
	if trimmed[0] == '[' {
		return looksLikeJSONArray(trimmed)
	}

	for i := 1; i < len(trimmed); i++ {
		switch trimmed[i] {
		case ' ', '\n', '\r', '\t':
			continue
		case '"', '}':
			return true
		default:
			return false
		}
	}
	return true
}

// looksLikeJSONArray 判断数组首个元素是否像合法 JSON value 的开头。
func looksLikeJSONArray(trimmed string) bool {
	for i := 1; i < len(trimmed); i++ {
		switch trimmed[i] {
		case ' ', '\n', '\r', '\t':
			continue
		case '{', '[', '"', ']', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return true
		case 't':
			return strings.HasPrefix(trimmed[i:], "true")
		case 'f':
			return strings.HasPrefix(trimmed[i:], "false")
		case 'n':
			return strings.HasPrefix(trimmed[i:], "null")
		default:
			return false
		}
	}
	return true
}

func isJSONStart(ch byte) bool {
	return ch == '{' || ch == '['
}

func matchingEnd(ch byte) byte {
	if ch == '[' {
		return ']'
	}
	return '}'
}

// extractJSONByBracketMatching 使用括号匹配算法提取 JSON
func extractJSONByBracketMatching(text string) (string, error) {
	// 找到第一个有效的起始字符
	start := -1

	for i := 0; i < len(text); i++ {
		if isJSONStart(text[i]) {
			start = i
			break
		}
	}

	if start == -1 {
		return "", fmt.Errorf("未找到 JSON 起始字符")
	}

	if candidate, ok := matchCompleteJSON(text, start); ok {
		return candidate, nil
	}

	return completePartialJSON(text[start:]), fmt.Errorf("未找到完整的 JSON 匹配")
}
