// Package render 提供用于处理和美化模型输出内容的渲染辅助工具。
package render

import (
	"strings"
)

// SplitThink 将原始文本内容拆分为“思考过程”和“最终回复”两部分。
// 返回值:
// - think: 提取出的思考过程内容（不含标签）
// - response: 移除思考标签后的回复内容
// - found: 是否在文本中找到了思考标签
func SplitThink(content string) (think, response string, found bool) {
	startIdx := strings.Index(content, "<think>")
	if startIdx == -1 {
		return "", content, false
	}

	endIdx := strings.Index(content[startIdx+7:], "</think>")
	if endIdx == -1 {
		return "", content, false
	}
	endIdx += startIdx + 7

	// 优化：避免使用 regexp.MustCompile 和 ReplaceAllString
	// 使用 strings.Index 和 strings.Builder 实现零分配的标签剥离
	var responseBuilder strings.Builder
	// 预分配大致容量，避免中间扩容
	responseBuilder.Grow(len(content) - (endIdx + 8 - startIdx))

	remaining := content
	firstThinkFound := false

	for {
		sIdx := strings.Index(remaining, "<think>")
		if sIdx == -1 {
			responseBuilder.WriteString(remaining)
			break
		}

		eIdx := strings.Index(remaining[sIdx+7:], "</think>")
		if eIdx == -1 {
			responseBuilder.WriteString(remaining)
			break
		}

		eIdx += sIdx + 7

		if !firstThinkFound {
			think = strings.TrimSpace(remaining[sIdx+7 : eIdx])
			firstThinkFound = true
			found = true
		}

		responseBuilder.WriteString(remaining[:sIdx])
		remaining = remaining[eIdx+8:]
	}

	return think, strings.TrimSpace(responseBuilder.String()), true
}
