package tool

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 用户提出的真实场景：前面有大量中文说明 + ```json 代码块包裹 + reason 字段被截断
const truncatedVulnReport = "基于对修复commit diff的深入分析和代码历史追踪，我现在可以确定该漏洞的影响范围。\\n\\n## 分析总结\\n\\n**漏洞本质**：在x86_64架构上，`#SS`（Stack Segment）异常处理使用了IST（Interrupt Stack Table）。\\n\\n**主线修复**：commit `6f442be2fb22` 合入于 `v3.18-rc6`。\\n\\n```json\\n{\"status\": \"success\", \"begin\": \"v2.6.12\", \"end\": \"v3.18-rc6\", \"fixed\": {\"3.2\": \"3.2.65\", \"3.4\": \"3.4.106\", \"3.10\": \"3.10.62\", \"3.12\": \"3.12.35\", \"3.14\": \"3.14.26\", \"3.16\": \"3.16.35\", \"3.17\": \"3.17.5\", \"3.18\": \"3.18-rc6\", \"3.19\": \"3.19-rc1\"}, \"reason\": \"漏洞代码（使用IST/paranoid处理#SS异常）从x86_64架构支持之初（v2.6.12可确认）即存在。修复commit 6f442be2fb22在v3.18-rc6合入主线。3.0、3.1、3.3、3.5、3.11、3.13、3.15等EOL分支未收到backport修复，所有版本均受影响"

func TestGetJsonText_CompleteValidJSON(t *testing.T) {
	input := `{"status": "success", "count": 42}`
	got, err := GetJsonText(input)
	require.NoError(t, err)
	assert.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, "success", m["status"])
}

func TestGetJsonText_JSONEmbeddedInText(t *testing.T) {
	input := "这里是一些中文前缀说明。\n\n结果如下：\n" +
		`{"status": "success", "begin": "v1.0", "end": "v2.0"}` +
		"\n\n后面还有一些描述文字。"

	got, err := GetJsonText(input)
	require.NoError(t, err)
	assert.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, "success", m["status"])
	assert.Equal(t, "v1.0", m["begin"])
}

func TestGetJsonText_NestedObject(t *testing.T) {
	input := "输出：" +
		`{"a": 1, "nested": {"b": 2, "c": {"d": 3}}, "e": "结束"}` +
		"done"

	got, err := GetJsonText(input)
	require.NoError(t, err)
	assert.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.InDelta(t, 1.0, m["a"], 0.0001)
}

func TestGetJsonText_ArrayJSON(t *testing.T) {
	input := "列表结果：" + `[{"id": 1}, {"id": 2}, {"id": 3}]` + "。"

	got, err := GetJsonText(input)
	require.NoError(t, err)
	assert.True(t, json.Valid([]byte(got)))

	var arr []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &arr))
	assert.Len(t, arr, 3)
}

// 核心修复场景：JSON 字符串未闭合 + 外层对象未闭合
func TestGetJsonText_TruncatedReasonField(t *testing.T) {
	got, err := GetJsonText(truncatedVulnReport)
	require.NoError(t, err, "截断的 JSON 应被括号匹配补齐后成功返回")
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))

	assert.Equal(t, "success", m["status"])
	assert.Equal(t, "v2.6.12", m["begin"])
	assert.Equal(t, "v3.18-rc6", m["end"])

	fixed, ok := m["fixed"].(map[string]interface{})
	require.True(t, ok, "fixed 字段应为嵌套对象")
	assert.Equal(t, "3.2.65", fixed["3.2"])
	assert.Equal(t, "3.19-rc1", fixed["3.19"])

	reason, ok := m["reason"].(string)
	require.True(t, ok, "reason 字段即使被截断也应被补齐为字符串")
	assert.Contains(t, reason, "漏洞代码")
}

// JSON 对象结构完整但 reason 字符串仍未闭合
func TestGetJsonText_UnclosedStringOnly(t *testing.T) {
	input := "前缀文本：" + `{"status": "ok", "msg": "这段话没有结尾`

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, "ok", m["status"])
	assert.Contains(t, m["msg"], "这段话没有结尾")
}

// 多层嵌套且末尾被截断
func TestGetJsonText_DeeplyNestedTruncated(t *testing.T) {
	input := "结果：" + `{"a": {"b": {"c": {"d": "未闭合的深层值`

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)), "补齐后应为合法 JSON: %s", got)

	// 至少能解析出顶层 a 字段
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	_, hasA := m["a"]
	assert.True(t, hasA)
}

// 字符串中包含转义引号不应影响匹配
func TestGetJsonText_StringWithEscapedQuotes(t *testing.T) {
	input := "输出：" + `{"cmd": "echo \"hello\"", "code": 0}` + " 完成"

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, `echo "hello"`, m["cmd"])
}

// 字符串内包含花括号不应干扰栈计数
func TestGetJsonText_BracesInsideString(t *testing.T) {
	input := `{"tpl": "hello {name}, welcome to {place}", "ok": true}`

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, true, m["ok"])
}

// 前面出现代码块里的花括号时，应跳过伪 JSON，继续提取后面的最终 JSON
func TestGetJsonText_SkipsCodeBracesBeforeFinalJSON(t *testing.T) {
	input := "分析过程：\n```c\nif (x) { return 1; }\n```\n最终结果：" +
		`{"status":"ok","items":[{"id":1,"name":"alpha"}]}`

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, "ok", m["status"])
}

// 普通说明中的方括号不应被 jsonrepair 抢先修成 JSON 数组
func TestGetJsonText_SkipsPlainBracketsBeforeFinalJSON(t *testing.T) {
	input := "候选字段 [not json] 仅用于说明，最终结果：" + `{"ok":true,"count":2}`

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, true, m["ok"])
}

// 明确标记为 json 的代码块支持根数组，兼容 LLM 直接返回列表结果的场景
func TestGetJsonText_JSONCodeBlockRootArray(t *testing.T) {
	input := "最终结果：\n```json\n" +
		`[{"type":"result","items":[{"id":1,"name":"alpha"}]}]` +
		"\n```"

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var arr []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &arr))
	require.Len(t, arr, 1)
	assert.Equal(t, "result", arr[0]["type"])
}

// 混合嵌套对象和数组时，括号匹配必须按真实配对关系出栈
func TestExtractJSONByBracketMatching_MixedObjectArray(t *testing.T) {
	input := `前缀 {"items":[{"id":1,"tags":["a","b"]}],"ok":true} 后缀`

	got, err := extractJSONByBracketMatching(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, true, m["ok"])
}

// 截断响应中同时存在对象和数组时，应补齐正确的混合括号
func TestGetJsonText_TruncatedMixedObjectArray(t *testing.T) {
	input := `结果：{"items":[{"msg":"内容被截断`

	got, err := GetJsonText(input)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	items, ok := m["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)
}

// 没有任何 JSON 起始字符时，会由 jsonrepair 兜底为合法 JSON（通常是一个字符串）
func TestGetJsonText_NoJSONStartChar(t *testing.T) {
	input := "这只是一段普通文本，没有任何 JSON 结构。"
	got, err := GetJsonText(input)
	// 允许成功（兜底修复）或失败，但若成功则必须是合法 JSON
	if err == nil {
		assert.True(t, json.Valid([]byte(got)), "兜底结果应为合法 JSON")
	}
}

// 空字符串
func TestGetJsonText_EmptyInput(t *testing.T) {
	_, err := GetJsonText("")
	assert.Error(t, err)
}

// 直接验证括号匹配在截断场景下的补齐行为
func TestExtractJSONByBracketMatching_PadsMissingBrackets(t *testing.T) {
	input := `{"a": {"b": "未闭合`
	partial, err := extractJSONByBracketMatching(input)
	// 未找到完整匹配会返回错误，但 partial 应当是补齐过的
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(partial, `{"a":`))
	// 末尾应补齐了字符串闭合引号和两个花括号
	assert.True(t, strings.HasSuffix(partial, `}}`))
	assert.Contains(t, partial, `"未闭合"`)
}

// LLM 结果被二次 JSON-encode：整体被引号包裹、内部 "/\n 被转义，包含 ```json 代码块
func TestGetJsonText_JSONEncodedWrapperWithCodeFence(t *testing.T) {
	// 构造原始文本（含真实换行和未转义的引号）
	inner := "现在我已经有了所有需要的数据。让我分析 Agent 与 NVD 之间的差异：\n\n" +
		"**关键差异：**\n1. **begin**: Agent 明确 `2.6.12` 作为引入版本\n\n" +
		"```json\n" +
		"{\n  \"items\": [\n" +
		"    {\"version\": \"v2.6.11\", \"expected_affected\": false},\n" +
		"    {\"version\": \"v2.6.12\", \"expected_affected\": true},\n" +
		"    {\"version\": \"v3.18-rc6\", \"expected_affected\": false}\n" +
		"  ]\n}\n" +
		"```"

	// 对原始文本进行 JSON-encode（模拟二次序列化的场景）
	encoded, err := json.Marshal(inner)
	require.NoError(t, err)

	got, err := GetJsonText(string(encoded))
	require.NoError(t, err, "被 JSON-encode 包裹的内容应先解码再提取")
	require.True(t, json.Valid([]byte(got)), "返回结果应是合法 JSON: %s", got)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))

	items, ok := m["items"].([]interface{})
	require.True(t, ok, "应提取出 items 数组而非整体字符串")
	assert.Len(t, items, 3)

	first := items[0].(map[string]interface{})
	assert.Equal(t, "v2.6.11", first["version"])
	assert.Equal(t, false, first["expected_affected"])
}

// 整体被引号包裹 + 截断的 JSON 也应能被提取
func TestGetJsonText_JSONEncodedWithTruncation(t *testing.T) {
	inner := "前缀文本\n```json\n{\"status\": \"ok\", \"msg\": \"内容被截断"
	encoded, err := json.Marshal(inner)
	require.NoError(t, err)

	got, err := GetJsonText(string(encoded))
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(got)))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(got), &m))
	assert.Equal(t, "ok", m["status"])
	assert.Contains(t, m["msg"], "内容被截断")
}

// 纯 JSON 字符串值（不包含嵌套的 { 或 [）应保持原来的完整验证行为，
// 后续则由 jsonrepair 处理。这里主要验证不会卡在无限递归或报错
func TestGetJsonText_PlainQuotedString(t *testing.T) {
	input := `"这只是一段普通字符串值"`
	got, err := GetJsonText(input)
	if err == nil {
		assert.True(t, json.Valid([]byte(got)))
	}
}
