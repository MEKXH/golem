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
	const startTag = "<think>"
	const endTag = "</think>"

	startIdx := strings.Index(content, startTag)
	if startIdx == -1 {
		return "", content, false
	}

	endIdx := strings.Index(content[startIdx+len(startTag):], endTag)
	if endIdx == -1 {
		return "", content, false
	}
	endIdx += startIdx + len(startTag)

	// Extract the think content
	think = strings.TrimSpace(content[startIdx+len(startTag) : endIdx])

	// Fast path: if the entire string is just the think block
	if startIdx == 0 && endIdx+len(endTag) == len(content) {
		return think, "", true
	}

	// Remove all <think>...</think> blocks from the response
	var respBuilder strings.Builder
	respBuilder.Grow(len(content) - (endIdx - startIdx))

	current := content
	for {
		sIdx := strings.Index(current, startTag)
		if sIdx == -1 {
			respBuilder.WriteString(current)
			break
		}

		eIdx := strings.Index(current[sIdx+len(startTag):], endTag)
		if eIdx == -1 {
			respBuilder.WriteString(current)
			break
		}
		eIdx += sIdx + len(startTag)

		respBuilder.WriteString(current[:sIdx])
		current = current[eIdx+len(endTag):]
	}

	return think, strings.TrimSpace(respBuilder.String()), true
}
