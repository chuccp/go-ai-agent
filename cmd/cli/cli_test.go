package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
)

// projectRoot 从测试文件路径向上找到项目根目录。
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// newTestCommand 加载 application.yml 并初始化 Command，返回可用的 Command。
func newTestCommand(t *testing.T) *Command {
	t.Helper()

	cfgPath := filepath.Join(projectRoot(), "application.yml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败 %s: %v", cfgPath, err)
	}

	ctx := core.NewContext(cfg, context.Background())
	cmd := &Command{}
	if err := cmd.Init(ctx); err != nil {
		t.Fatalf("Command.Init 失败: %v", err)
	}
	return cmd
}

func TestFullChatFlow(t *testing.T) {
	cmd := newTestCommand(t)

	// 发送一条简单消息
	sent := cmd.HandleMessage("评价一个 凡人修仙传？不要调用工具。")
	if !sent {
		t.Fatal("HandleMessage 返回 false，消息发送失败")
	}

	// 阻塞读取流式响应直到 done
	result := readUntilDone(t, cmd)

	if result == "" {
		t.Fatal("AI 返回空响应")
	}
	if strings.Contains(result, "[Error]") {
		t.Fatalf("AI 返回错误: %s", result)
	}

	t.Logf("✅ 完整对话流程通过\n   AI 回复: %s", result)
}

func TestToolCalling(t *testing.T) {
	cmd := newTestCommand(t)

	// 发送需要调用工具的消息
	sent := cmd.HandleMessage("用 execute_command 工具执行命令 ls -la，然后告诉我看到了什么。命令结果用中文简短说明即可。")
	if !sent {
		t.Fatal("HandleMessage 返回 false，消息发送失败")
	}

	result := readUntilDone(t, cmd)

	if result == "" {
		t.Fatal("AI 返回空响应，工具可能未被调用")
	}
	if strings.Contains(result, "[Error]") {
		t.Fatalf("工具调用返回错误: %s", result)
	}

	t.Logf("✅ 工具调用流程通过\n   AI 回复: %s", result)
}

func TestMultipleMessages(t *testing.T) {
	cmd := newTestCommand(t)

	rounds := []struct {
		msg      string
		mustHave string // 回复中必须包含的关键词，用于验证上下文
		mustNot  string // 回复中禁止出现的内容
		desc     string
	}{
		{
			msg:      "请记住：我最喜欢的编程语言是 Go。只需要回复 已记住 。",
			mustHave: "已记住",
			mustNot:  "",
			desc:     "第一轮（设置上下文）",
		},
		{
			msg:      "我刚才说我最喜欢的编程语言是什么？一句话回答。",
			mustHave: "Go",
			mustNot:  "",
			desc:     "第二轮（验证上下文）",
		},
		{
			msg:      "把它的名字全部改成大写告诉我。",
			mustHave: "GO",
			mustNot:  "",
			desc:     "第三轮（深度引用上下文）",
		},
	}

	for _, round := range rounds {
		sent := cmd.HandleMessage(round.msg)
		if !sent {
			t.Fatalf("[%s] HandleMessage 返回 false", round.desc)
		}

		result := readUntilDone(t, cmd)
		if result == "" {
			t.Fatalf("[%s] AI 返回空响应", round.desc)
		}
		if strings.Contains(result, "[Error]") {
			t.Fatalf("[%s] AI 返回错误: %s", round.desc, result)
		}
		if round.mustHave != "" && !strings.Contains(result, round.mustHave) {
			t.Fatalf("[%s] 回复中缺少关键词 %q，上下文丢失。\n实际回复: %s", round.desc, round.mustHave, result)
		}
		if round.mustNot != "" && strings.Contains(result, round.mustNot) {
			t.Fatalf("[%s] 回复中包含禁止内容 %q。\n实际回复: %s", round.desc, round.mustNot, result)
		}

		t.Logf("[%s] ✅ %s", round.desc, result)
	}

	t.Log("✅ 多轮对话流程通过")
}

// readUntilDone 阻塞读取事件直到收到 done，返回累积的完整文本。
func readUntilDone(t *testing.T, cmd *Command) string {
	t.Helper()

	var result strings.Builder
	deadline := time.After(60 * time.Second)

	for {
		select {
		case <-deadline:
			t.Fatal("读取响应超时（60s）")
			return ""
		default:
		}

		event := cmd.ReadEvent()
		if event == nil {
			// 流中断（队列关闭等异常情况）
			return result.String()
		}

		switch event.Type {
		case agent.EventTypeChunk:
			result.WriteString(event.Content)
			fmt.Print(event.Content) // 实时输出到控制台
		case agent.EventTypeError:
			t.Logf("\n❌ Error: %s", event.Message)
			return "[Error] " + event.Message
		case agent.EventTypeDone:
			fmt.Println() // 换行
			return result.String()
		}
	}
}
