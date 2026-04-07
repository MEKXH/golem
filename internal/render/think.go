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

	think = strings.TrimSpace(content[startIdx+len(startTag) : endIdx])

	// If there are no other start tags, we can optimize without strings.Builder
	remainingStartIdx := strings.Index(content[endIdx+len(endTag):], startTag)
	if remainingStartIdx == -1 {
		response = strings.TrimSpace(content[:startIdx] + content[endIdx+len(endTag):])
		return think, response, true
	}

	var sb strings.Builder
	sb.WriteString(content[:startIdx])

	remaining := content[endIdx+len(endTag):]
	for {
		s := strings.Index(remaining, startTag)
		if s == -1 {
			sb.WriteString(remaining)
			break
		}
		e := strings.Index(remaining[s+len(startTag):], endTag)
		if e == -1 {
			sb.WriteString(remaining)
			break
		}
		e += s + len(startTag)

		sb.WriteString(remaining[:s])
		remaining = remaining[e+len(endTag):]
	}

	response = strings.TrimSpace(sb.String())
	return think, response, true
}
