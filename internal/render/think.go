// Package render 提供用于处理和美化模型输出内容的渲染辅助工具。
package render

import (
	"strings"
)

// SplitThink 将原始文本内容拆分为“思考过程”和“最终回复”两部分。
// 它通过 strings.Index 和 strings.Builder 来实现零分配（Zero-allocation）解析，
// 避免了正则表达式在长文本场景下的严重性能开销。
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

	var builder strings.Builder
	builder.Grow(len(content))

	current := content

	for {
		idx := strings.Index(current, startTag)
		if idx == -1 {
			// 未找到开始标签，直接追加剩余内容
			builder.WriteString(current)
			break
		}

		endIdx := strings.Index(current[idx+len(startTag):], endTag)
		if endIdx == -1 {
			// 存在开始标签但未闭合，按照非贪婪正则表达式的行为，保留剩余所有内容（包括未闭合的标签）
			builder.WriteString(current)
			break
		}

		// 只提取第一个完整的 think block 内容
		if !found {
			think = strings.TrimSpace(current[idx+len(startTag) : idx+len(startTag)+endIdx])
			found = true
		}

		// 写入该 think block 之前的内容
		builder.WriteString(current[:idx])
		// 移动当前游标到结束标签之后
		current = current[idx+len(startTag)+endIdx+len(endTag):]
	}

	if found {
		response = strings.TrimSpace(builder.String())
		return think, response, true
	}

	return "", content, false
}
