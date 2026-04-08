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

	think = strings.TrimSpace(content[startIdx+7 : endIdx])
	response = strings.TrimSpace(content[:startIdx] + content[endIdx+8:])

	return think, response, true
}
