// Package render 提供用于处理和美化模型输出内容的渲染辅助工具。
package render

import (
	"regexp"
	"strings"
)

// thinkBlockRe 是用于匹配并提取模型输出中 <think> 标签内容的正则表达式。
var thinkBlockRe = regexp.MustCompile(`(?s)<think>(.*?)</think>`)

// SplitThink 将原始文本内容拆分为“思考过程”和“最终回复”两部分。
// 返回值:
// - think: 提取出的思考过程内容（不含标签）
// - response: 移除思考标签后的回复内容
// - found: 是否在文本中找到了思考标签
func SplitThink(content string) (think, response string, found bool) {
	matches := thinkBlockRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		think = strings.TrimSpace(matches[1])
		response = strings.TrimSpace(thinkBlockRe.ReplaceAllString(content, ""))
		return think, response, true
	}
	return "", content, false
}
