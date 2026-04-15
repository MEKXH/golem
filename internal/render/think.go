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
	sIdx := strings.Index(content, "<think>")
	if sIdx == -1 {
		return "", content, false
	}

	var b strings.Builder
	b.Grow(len(content)) // 预分配以减少内存分配
	curr := content
	foundThink := false

	for {
		s := strings.Index(curr, "<think>")
		if s == -1 {
			b.WriteString(curr)
			break
		}

		e := strings.Index(curr[s+7:], "</think>")
		if e == -1 {
			b.WriteString(curr)
			break
		}
		e += s + 7

		if !foundThink {
			think = strings.TrimSpace(curr[s+7 : e])
			foundThink = true
		}

		b.WriteString(curr[:s])
		curr = curr[e+8:]
	}

	if !foundThink {
		return "", content, false
	}

	return think, strings.TrimSpace(b.String()), true
}
