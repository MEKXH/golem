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
//
// ⚡ Bolt Optimization: Uses strings.Index and strings.Builder for zero-allocation HTML parsing,
// avoiding the overhead of the regex engine and intermediate string allocations.
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

	var sb strings.Builder
	sb.Grow(len(content))
	curr := content
	var firstThink string

	for {
		sIdx := strings.Index(curr, startTag)
		if sIdx == -1 {
			sb.WriteString(curr)
			break
		}

		eIdx := strings.Index(curr[sIdx+len(startTag):], endTag)
		if eIdx == -1 {
			sb.WriteString(curr)
			break
		}
		eIdx += sIdx + len(startTag)

		if !found {
			firstThink = curr[sIdx+len(startTag) : eIdx]
			found = true
		}

		sb.WriteString(curr[:sIdx])
		curr = curr[eIdx+len(endTag):]
	}

	if found {
		return strings.TrimSpace(firstThink), strings.TrimSpace(sb.String()), true
	}

	return "", content, false
}
