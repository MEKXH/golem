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

	firstStart := strings.Index(content, startTag)
	if firstStart == -1 {
		return "", content, false
	}

	var firstThink string
	var firstFound bool

	var builder strings.Builder
	builder.Grow(len(content))

	curr := content
	for {
		start := strings.Index(curr, startTag)
		if start == -1 {
			builder.WriteString(curr)
			break
		}

		// Found start tag, append everything before it
		builder.WriteString(curr[:start])

		// Move past start tag
		afterStart := curr[start+len(startTag):]

		end := strings.Index(afterStart, endTag)
		if end == -1 {
			// No closing tag found.
			builder.WriteString(startTag)
			builder.WriteString(afterStart)
			break
		}

		// Found closing tag. Extract the block.
		thinkBlock := afterStart[:end]
		if !firstFound {
			firstThink = thinkBlock
			firstFound = true
		}

		// Move past the closing tag
		curr = afterStart[end+len(endTag):]
	}

	if !firstFound {
		// Only happened if start was found but no end
		return "", content, false
	}

	return strings.TrimSpace(firstThink), strings.TrimSpace(builder.String()), true
}
