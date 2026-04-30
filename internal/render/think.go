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

	afterStart := content[startIdx+len("<think>"):]
	endIdx := strings.Index(afterStart, "</think>")

	if endIdx == -1 {
		return "", content, false
	}

	think = strings.TrimSpace(afterStart[:endIdx])

	var responseBuilder strings.Builder
	responseBuilder.Grow(len(content) - len("<think>") - len("</think>") - endIdx)

	responseBuilder.WriteString(content[:startIdx])

	remaining := afterStart[endIdx+len("</think>"):]

	for {
		nextStartIdx := strings.Index(remaining, "<think>")
		if nextStartIdx == -1 {
			responseBuilder.WriteString(remaining)
			break
		}

		responseBuilder.WriteString(remaining[:nextStartIdx])

		nextAfterStart := remaining[nextStartIdx+len("<think>"):]
		nextEndIdx := strings.Index(nextAfterStart, "</think>")

		if nextEndIdx == -1 {
			responseBuilder.WriteString(remaining[nextStartIdx:])
			break
		}

		remaining = nextAfterStart[nextEndIdx+len("</think>"):]
	}

	return think, strings.TrimSpace(responseBuilder.String()), true
}
