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
// 优化: 避免使用正则表达式和大量内存分配
func SplitThink(content string) (think, response string, found bool) {
	startTag := "<think>"
	endTag := "</think>"

	firstStart := strings.Index(content, startTag)
	if firstStart == -1 {
		return "", content, false
	}

	firstEnd := strings.Index(content[firstStart+len(startTag):], endTag)
	if firstEnd == -1 {
		return "", content, false
	}

	firstEnd += firstStart + len(startTag)
	think = strings.TrimSpace(content[firstStart+len(startTag) : firstEnd])

	var b strings.Builder
	b.Grow(len(content))

	curr := 0
	for {
		start := strings.Index(content[curr:], startTag)
		if start == -1 {
			b.WriteString(content[curr:])
			break
		}

		end := strings.Index(content[curr+start+len(startTag):], endTag)
		if end == -1 {
			b.WriteString(content[curr:])
			break
		}

		b.WriteString(content[curr : curr+start])
		curr = curr + start + len(startTag) + end + len(endTag)
	}

	return think, strings.TrimSpace(b.String()), true
}
