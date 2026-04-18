// Package render 提供用于处理和美化模型输出内容的渲染辅助工具。
package render

import (
	"strings"
)

// SplitThink 将原始文本内容拆分为“思考过程”和“最终回复”两部分。
// 它通过基于字符串查找的单遍遍历代替正则表达式，实现了近乎零内存分配（zero-allocation）的高性能解析，
// 并从原始输出中彻底剔除所有的 <think> 块，将第一个提取出的思考内容作为返回值。
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

	var builder strings.Builder
	// 预分配可能的最大容量，减少内部扩容时的分配次数
	builder.Grow(len(content) - (endIdx + 15))
	var firstThink string

	curr := content
	for {
		sIdx := strings.Index(curr, "<think>")
		if sIdx == -1 {
			builder.WriteString(curr)
			break
		}

		eIdx := strings.Index(curr[sIdx+7:], "</think>")
		if eIdx == -1 {
			builder.WriteString(curr)
			break
		}

		builder.WriteString(curr[:sIdx])
		if !found {
			firstThink = curr[sIdx+7 : sIdx+7+eIdx]
			found = true
		}
		// 移动指针到 </think> 之后 (7 为 "<think>" 长度, 8 为 "</think>" 长度)
		curr = curr[sIdx+7+eIdx+8:]
	}

	return strings.TrimSpace(firstThink), strings.TrimSpace(builder.String()), true
}
