package tools

import (
	"context"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"testing"
)

type dummyTool struct {
	name string
}

func (t *dummyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name, Desc: "desc"}, nil
}

func (t *dummyTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	return "ok", nil
}

func BenchmarkGetToolInfos(b *testing.B) {
	reg := NewRegistry()
	for i := 0; i < 15; i++ {
		reg.Register(&dummyTool{name: "tool" + string(rune(i))})
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.GetToolInfos(ctx)
	}
}
