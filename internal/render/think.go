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

	var thinkStr string
	var respBuilder strings.Builder
	respBuilder.Grow(len(content))

	curr := content
	firstMatch := true

	for {
		sIdx := strings.Index(curr, startTag)
		if sIdx == -1 {
			respBuilder.WriteString(curr)
			break
		}

		eIdx := strings.Index(curr[sIdx+len(startTag):], endTag)
		if eIdx == -1 {
			respBuilder.WriteString(curr)
			break
		}

		found = true
		respBuilder.WriteString(curr[:sIdx])

		if firstMatch {
			thinkStr = curr[sIdx+len(startTag) : sIdx+len(startTag)+eIdx]
			firstMatch = false
		}

		curr = curr[sIdx+len(startTag)+eIdx+len(endTag):]
	}

	if !found {
		return "", content, false
	}

	return strings.TrimSpace(thinkStr), strings.TrimSpace(respBuilder.String()), true
}
