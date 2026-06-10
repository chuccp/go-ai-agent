package runner

import (
	"context"
	"github.com/bytedance/sonic"
	"fmt"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/chat"
	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-ai-agent/flow/cache"
	"github.com/chuccp/go-ai-agent/flow/engine"
	"github.com/chuccp/go-ai-agent/entity"
	flowModel "github.com/chuccp/go-ai-agent/model"
	flownodes "github.com/chuccp/go-ai-agent/flow/nodes"

	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// FlowRunner 流程执行器，管理流程的 WebSocket 交互执行
type FlowRunner struct {
	ctx            *core.Context
	chatService    *chat.UnifiedChatService
	flowModel      *flowModel.FlowModel
	nodeModel      *flowModel.FlowNodeModel
	edgeModel      *flowModel.FlowEdgeModel
	executionModel *flowModel.FlowExecutionModel
	registry       *engine.Registry // 节点类型注册表（初始化一次）
	taskMgr        *engine.TaskManager
	functions      *engine.FunctionRegistry
	cacheMgr       *cache.CacheManager

	running map[uint]*runningExecution
	mu      sync.Mutex

	sendFn func(data []byte)
}

type runningExecution struct {
	engine        *engine.Engine
	ctx           *engine.ExecutionContext
	currentNodeId uint
}

func NewFlowRunner() *FlowRunner {
	return &FlowRunner{
		running: make(map[uint]*runningExecution),
	}
}

func (r *FlowRunner) Init(ctx *core.Context) error {
	r.ctx = ctx
	r.chatService = core.GetService[*chat.UnifiedChatService](ctx)
	r.flowModel = core.GetModel[*flowModel.FlowModel](ctx)
	r.nodeModel = core.GetModel[*flowModel.FlowNodeModel](ctx)
	r.edgeModel = core.GetModel[*flowModel.FlowEdgeModel](ctx)
	r.executionModel = core.GetModel[*flowModel.FlowExecutionModel](ctx)

	// 节点注册表
	r.registry = engine.NewRegistry()
	r.registry.Register(&flownodes.StartNode{})
	r.registry.Register(&flownodes.EndNode{})
	r.registry.Register(flownodes.NewLLMNode())
	r.registry.Register(&flownodes.UserInputNode{})
	r.registry.Register(flownodes.NewForEachNode())
	r.registry.Register(flownodes.NewSplitNode())
	r.registry.Register(flownodes.NewTransformNode())
	r.registry.Register(flownodes.NewConditionNode())
	r.registry.Register(flownodes.NewScriptNode())
	r.registry.Register(flownodes.NewIteratorNode())
	r.registry.Register(flownodes.NewLoopNode())
	r.registry.Register(flownodes.NewImageGenNode())
	r.registry.Register(flownodes.NewAudioGenNode())
	r.registry.Register(flownodes.NewVideoGenNode())

	// 任务管理器
	r.taskMgr = engine.NewTaskManager(4)

	// 函数注册表
	r.functions = engine.NewFunctionRegistry()
	r.registerFunctions()

	// 缓存管理器
	r.cacheMgr = cache.NewCacheManager("./data/cache", true)

	log.Info("FlowRunner 已初始化")
	return nil
}

func (r *FlowRunner) registerFunctions() {
	// llm 函数
	r.functions.Register("llm", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		model, _ := args["model"].(string)
		if model == "" {
			model = r.chatService.GetDefaultPath()
		}
		prompt, _ := args["prompt"].(string)
		system, _ := args["system"].(string)
		maxTokens := 4096
		if mt, ok := args["max_tokens"].(float64); ok && mt > 0 {
			maxTokens = int(mt)
		}
		jsonMode, _ := args["json_mode"].(bool)
		stream, _ := args["stream"].(bool)

		var history []common.ChatMessage
		if system != "" {
			history = append(history, common.ChatMessage{Role: "system", Content: system})
		}

		opts := common.NewLLMOptions()
		opts.SetMaxTokens(maxTokens)
		if jsonMode {
			opts.SetJSONMode(true)
		}

		if stream {
			handler := common.NewStreamHandler()
			var fullText strings.Builder
			handler.OnContent(func(content string, reasoning bool) {
				if !reasoning {
					fullText.WriteString(content)
				}
				if ctx.Emitter != nil {
					ctx.Emitter.Emit(engine.FlowEvent{
						Type:        engine.EventNodeChunk,
						ExecutionId: ctx.ExecutionId,
						Content:     content,
					})
				}
			})

			err := r.chatService.ChatStreamWithContext(context.Background(), model, history, prompt, handler, opts)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"output": fullText.String(),
				"prompt": prompt,
			}, nil
		}

		// 非流式：用于 for_each / iterator 等内部调用
		result, err := r.chatService.ChatWithHistoryWithContext(context.Background(), model, history, prompt, opts)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"output": result,
			"prompt": prompt,
		}, nil
	})

	// image_generation 函数
	r.functions.Register("image_generation", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		// TODO: 接入图片生成 provider
		return map[string]any{
			"output": "",
			"urls":   []string{},
			"count":  0,
		}, fmt.Errorf("image generation not implemented yet")
	})

	// audio_generation 函数
	r.functions.Register("audio_generation", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		// TODO: 接入语音生成 provider
		return map[string]any{
			"output": "",
			"url":    "",
		}, fmt.Errorf("audio generation not implemented yet")
	})

	// video_generation 函数
	r.functions.Register("video_generation", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		// TODO: 接入视频生成 provider
		return map[string]any{
			"output": "",
			"url":    "",
		}, fmt.Errorf("video generation not implemented yet")
	})
}

// SetSendFunc 设置事件发送函数（由 ChatRunner 调用，注入 WS 能力）
func (r *FlowRunner) SetSendFunc(fn func(data []byte)) {
	r.sendFn = fn
}

func (r *FlowRunner) HandleFlowStart(flowId uint, executionId uint, sessionId uint, initialInput string) error {
	_, err := r.flowModel.FindById(flowId)
	if err != nil {
		return fmt.Errorf("flow not found: %w", err)
	}

	flowNodes, err := r.nodeModel.FindByFlowId(flowId)
	if err != nil {
		return fmt.Errorf("failed to load nodes: %w", err)
	}

	flowEdges, err := r.edgeModel.FindByFlowId(flowId)
	if err != nil {
		return fmt.Errorf("failed to load edges: %w", err)
	}

	var exec *entity.FlowExecution
	if executionId > 0 {
		exec, err = r.executionModel.FindById(executionId)
		if err != nil {
			return fmt.Errorf("execution not found: %w", err)
		}
	} else {
		exec = &entity.FlowExecution{
			FlowId:    flowId,
			SessionId: sessionId,
			Status:    engine.ExecRunning,
			Context:   "{}",
		}
		if err := r.executionModel.Create(exec); err != nil {
			return fmt.Errorf("failed to create execution: %w", err)
		}
		executionId = exec.Id
	}

	eng := engine.NewEngine(r.registry, r)
	eng.SetTaskManager(r.taskMgr)
	eng.LoadFlow(flowNodes, flowEdges)

	startId, err := eng.FindStartNode()
	if err != nil {
		return err
	}

	execCtx := engine.NewExecutionContext(flowId, executionId, sessionId, r)
	execCtx.Functions = r.functions
	execCtx.Cache = r.cacheMgr

	if initialInput != "" {
		execCtx.SeedInput(initialInput)
	}

	r.mu.Lock()
	r.running[executionId] = &runningExecution{
		engine:        eng,
		ctx:           execCtx,
		currentNodeId: startId,
	}
	r.mu.Unlock()

	exec.Status = engine.ExecRunning
	if err := r.executionModel.Update(exec); err != nil {
		log.Warn("Failed to update execution status", zap.Error(err))
	}

	go func() {
		err := eng.Run(execCtx, startId)
		r.mu.Lock()
		delete(r.running, executionId)
		r.mu.Unlock()

		if err != nil {
			exec.Status = engine.ExecError
			exec.Context = fmt.Sprintf(`{"error":"%s"}`, err.Error())
		} else {
			exec.Status = engine.ExecCompleted
			ctxJSON, _ := sonic.Marshal(map[string]any{
				"data":         execCtx.Data,
				"node_outputs": execCtx.NodeOutputs,
			})
			exec.Context = string(ctxJSON)
		}
		r.executionModel.Update(exec)

		log.Info("Flow execution finished",
			zap.Uint("executionId", executionId),
			zap.Uint("flowId", flowId),
			zap.String("status", exec.Status))
	}()

	return nil
}

func (r *FlowRunner) HandleUserResponse(executionId uint, response string) error {
	r.mu.Lock()
	re, ok := r.running[executionId]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("no running execution found for id %d", executionId)
	}

	re.ctx.SendUserInput(response)
	return nil
}

func (r *FlowRunner) HandleFlowStop(executionId uint) error {
	r.mu.Lock()
	re, ok := r.running[executionId]
	r.mu.Unlock()

	if !ok {
		if exec, err := r.executionModel.FindById(executionId); err == nil {
			exec.Status = engine.ExecError
			r.executionModel.Update(exec)
		}
		return fmt.Errorf("no running execution found for id %d", executionId)
	}

	re.ctx.Abort()

	if exec, err := r.executionModel.FindById(executionId); err == nil {
		exec.Status = engine.ExecError
		r.executionModel.Update(exec)
	}

	return nil
}

// Emit 实现 engine.EventEmitter 接口
func (r *FlowRunner) Emit(event engine.FlowEvent) {
	if r.sendFn == nil {
		return
	}
	data, err := sonic.Marshal(event)
	if err != nil {
		log.Error("序列化流程事件失败", zap.Error(err))
		return
	}
	r.sendFn(data)
}
