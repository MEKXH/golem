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
	startTag := "<think>"
	endTag := "</think>"

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

	nextStartIdx := strings.Index(content[endIdx+len(endTag):], startTag)
	if nextStartIdx == -1 {
		response = content[:startIdx] + content[endIdx+len(endTag):]
		return think, strings.TrimSpace(response), true
	}

	var builder strings.Builder
	builder.Grow(len(content) - (endIdx - startIdx + len(endTag)))
	builder.WriteString(content[:startIdx])

	currentIdx := endIdx + len(endTag)
	for {
		sIdx := strings.Index(content[currentIdx:], startTag)
		if sIdx == -1 {
			builder.WriteString(content[currentIdx:])
			break
		}
		sIdx += currentIdx

		eIdx := strings.Index(content[sIdx+len(startTag):], endTag)
		if eIdx == -1 {
			builder.WriteString(content[currentIdx:])
			break
		}
		eIdx += sIdx + len(startTag)

		builder.WriteString(content[currentIdx:sIdx])
		currentIdx = eIdx + len(endTag)
	}

	return think, strings.TrimSpace(builder.String()), true
}
