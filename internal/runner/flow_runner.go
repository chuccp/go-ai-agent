package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chuccp/go-ai-agent/internal/ai"
	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/flow/appstore"
	"github.com/chuccp/go-ai-agent/internal/flow/cache"
	"github.com/chuccp/go-ai-agent/internal/flow/engine"
	flownodes "github.com/chuccp/go-ai-agent/internal/flow/nodes"
	flowModel "github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/service"

	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// FlowRunner manages flow execution via WebSocket interaction
type FlowRunner struct {
	core.IRunner
	ctx            *core.Context
	chatService    *chat.UnifiedChatService
	genService     *ai.GenService
	flowService    *service.FlowService
	flowModel      *flowModel.FlowModel
	executionModel *flowModel.FlowExecutionModel
	appStore       *appstore.Store
	registry       *engine.Registry // Node type registry (initialized once)
	taskMgr        *engine.TaskManager
	functions      *engine.FunctionRegistry
	cacheMgr       *cache.CacheManager

	builtInFlows map[string]*BuiltInFlow // System-built-in flows, keyed by name

	running map[uint]*runningExecution
	mu      sync.Mutex

	sendFn func(data []byte)
}

// BuiltInFlow holds an in-memory flow definition used by the system itself.
type BuiltInFlow struct {
	Definition *entity.FlowDefinition
	Nodes      []*entity.FlowNode
	Edges      []*entity.FlowEdge
}

type runningExecution struct {
	engine        *engine.Engine
	ctx           *engine.ExecutionContext
	currentNodeId uint
	waitCh        chan string   // receives waiting prompts from user_input nodes
	doneCh        chan struct{} // closed when execution finishes
}

func NewFlowRunner() *FlowRunner {
	return &FlowRunner{
		running: make(map[uint]*runningExecution),
	}
}

func (r *FlowRunner) Init(ctx *core.Context) error {
	r.ctx = ctx
	r.chatService = core.GetService[*chat.UnifiedChatService](ctx)
	r.genService = core.GetService[*ai.GenService](ctx)
	r.flowService = core.GetService[*service.FlowService](ctx)
	r.flowModel = core.GetModel[*flowModel.FlowModel](ctx)
	r.executionModel = core.GetModel[*flowModel.FlowExecutionModel](ctx)
	if r.flowService != nil {
		r.appStore = r.flowService.GetAppStore()
	}

	// Node registry
	r.registry = engine.NewRegistry()
	r.registry.Register(&flownodes.StartNode{})
	r.registry.Register(&flownodes.EndNode{})
	r.registry.Register(flownodes.NewLLMNode())
	r.registry.Register(&flownodes.UserInputNode{})
	r.registry.Register(flownodes.NewForEachNode())
	r.registry.Register(flownodes.NewSplitNode())
	r.registry.Register(flownodes.NewTransformNode())
	r.registry.Register(flownodes.NewConditionNode())
	r.registry.Register(flownodes.NewSwitchNode())
	r.registry.Register(flownodes.NewExecuteNode())
	r.registry.Register(flownodes.NewScriptNode())
	r.registry.Register(flownodes.NewIteratorNode())
	r.registry.Register(flownodes.NewLoopNode())
	r.registry.Register(flownodes.NewImageGenNode())
	r.registry.Register(flownodes.NewAudioGenNode())
	r.registry.Register(flownodes.NewVideoGenNode())
	r.registry.Register(flownodes.NewSkillNode())

	// Internal nodes (not exposed to frontend)
	if r.flowService != nil {
		r.registry.Register(flownodes.NewCreateFlowNode(r.flowService))
	}

	// Built-in system flows
	r.builtInFlows = make(map[string]*BuiltInFlow)
	r.registerBuiltInFlows()

	// Task manager
	r.taskMgr = engine.NewTaskManager(4)

	// Function registry
	r.functions = engine.NewFunctionRegistry()
	r.registerFunctions()

	// Cache manager
	r.cacheMgr = cache.NewCacheManager("./data/cache", true)

	log.Info("FlowRunner initialized")
	return nil
}

// registerBuiltInFlows defines system flows that live in memory, not in the database.
func (r *FlowRunner) registerBuiltInFlows() {
	r.builtInFlows["create_flow"] = builtInCreateFlow(r.chatService.GetDefaultPath())
}

// builtInCreateFlow returns the "create flow" meta-flow definition.
// Structure: Start → UserInput → Loop[LLM ask → UserInput confirm] → LLM generate JSON → create_flow → End
func builtInCreateFlow(defaultModel string) *BuiltInFlow {
	if defaultModel == "" {
		defaultModel = "1.default"
	}

	const (
		startID     = 1
		inputID     = 2
		loopID      = 3
		loopLLMID   = 4
		loopInputID = 5
		genJSONID   = 6
		saveID      = 7
		endID       = 8
	)

	groupID := uint(loopID)

	nodes := []*entity.FlowNode{
		{Id: startID, Type: flownodes.TypeStart, Label: "start", Config: "{}", PositionX: 80, PositionY: 200},
		{Id: inputID, Type: flownodes.TypeUserInput, Label: "describe_flow", Config: `{"prompt":"请描述你想创建的流程：用来做什么？输入是什么？输出是什么？"}`, PositionX: 280, PositionY: 200},
		{
			Id: loopID, Type: flownodes.TypeLoop, Label: "design_loop",
			Config:    `{"max_iterations":5,"break_field":"ask_design.output","break_operator":"contains","break_value":"READY"}`,
			PositionX: 480, PositionY: 200,
		},
		{
			Id: loopLLMID, Type: flownodes.TypeLLM, Label: "ask_design", GroupId: &groupID,
			Config: mustJSON(map[string]any{
				"model":  defaultModel,
				"prompt": "你是流程设计助手。用户想创建一个新流程。\n用户需求：{{describe_flow.output}}\n如果信息还不够，请提出一个最需要的追问；如果已经足够，请直接回复 'READY' 并给出简短设计方案。",
			}),
			PositionX: 480, PositionY: 80,
		},
		{
			Id: loopInputID, Type: flownodes.TypeUserInput, Label: "user_response", GroupId: &groupID,
			Config:    `{"prompt":"请回复"}`,
			PositionX: 680, PositionY: 80,
		},
		{
			Id: genJSONID, Type: flownodes.TypeLLM, Label: "generate_json",
			Config: mustJSON(map[string]any{
				"model":      defaultModel,
				"max_tokens": 2000,
				"prompt":     "根据用户需求生成一个流程的 JSON 定义。\n需求：{{describe_flow.output}}\nLLM 与用户的确认对话：{{user_response.output}}\n\n请输出标准 JSON，字段如下：\n{\"name\":\"流程名称\",\"description\":\"描述\",\"category\":\"分类\",\"icon\":\"📖\",\"nodes\":[{\"type\":\"start\",\"label\":\"start\",\"config\":{},\"position_x\":100,\"position_y\":200}],\"edges\":[]}\n\n重要规则（必须遵守，否则流程无法运行）：\n1. 必须包含一个 type=\"user_input\"、label=\"user_input\" 的节点，用于接收用户输入的一句话。\n2. LLM 节点的 model 字段必须使用 'id.model' 格式，例如 '1.deepseek-v4-flash'，不要只写模型名。\n3. LLM 节点的 prompt 字段引用用户输入时，必须原样包含字面量模板占位符 `{{user_input.output}}`（包括双花括号）。运行时系统会自动把它替换为用户的实际输入。\n   错误示例（会导致运行失败）：{{user_input}}、{{node_0.output}}、把 {{user_input.output}} 替换成示例句子。\n   正确示例：\"prompt\":\"请根据以下句子写一篇300字的故事：{{user_input.output}}\"\n4. edges 使用 source/target（或 from/to）0-based 索引。\n5. 节点 label 必须使用英文或中文名称，prompt 模板中引用节点时必须使用 label，不能使用 node_0 这类索引。\n\n示例（一句话生成故事）：\n{\"name\":\"一句话故事生成\",\"description\":\"输入一句话，生成300字故事\",\"category\":\"写作\",\"icon\":\"📖\",\"nodes\":[{\"type\":\"start\",\"label\":\"start\",\"config\":{},\"position_x\":100,\"position_y\":200},{\"type\":\"user_input\",\"label\":\"user_input\",\"config\":{\"prompt\":\"请输入一句话\"},\"position_x\":300,\"position_y\":200},{\"type\":\"llm\",\"label\":\"generate_story\",\"config\":{\"model\":\"1.deepseek-v4-flash\",\"prompt\":\"请根据以下句子写一篇300字的故事：{{user_input.output}}\"},\"position_x\":500,\"position_y\":200},{\"type\":\"end\",\"label\":\"end\",\"config\":{},\"position_x\":700,\"position_y\":200}],\"edges\":[{\"source\":0,\"target\":1},{\"source\":1,\"target\":2},{\"source\":2,\"target\":3}]}\n\n只输出 JSON，不要解释。",
			}),
			PositionX: 680, PositionY: 200,
		},
		{
			Id: saveID, Type: flownodes.TypeCreateFlow, Label: "save_flow",
			Config:    `{"source":"generate_json"}`,
			PositionX: 880, PositionY: 200,
		},
		{Id: endID, Type: flownodes.TypeEnd, Label: "end", Config: "{}", PositionX: 1080, PositionY: 200},
	}

	edges := []*entity.FlowEdge{
		{SourceNodeId: startID, TargetNodeId: inputID, SourceHandle: "output", TargetHandle: "input"},
		{SourceNodeId: inputID, TargetNodeId: loopID, SourceHandle: "output", TargetHandle: "input"},
		{SourceNodeId: loopID, TargetNodeId: genJSONID, SourceHandle: "output", TargetHandle: "input"},
		{SourceNodeId: genJSONID, TargetNodeId: saveID, SourceHandle: "output", TargetHandle: "input"},
		{SourceNodeId: saveID, TargetNodeId: endID, SourceHandle: "output", TargetHandle: "input"},
		// loop body edges
		{SourceNodeId: loopLLMID, TargetNodeId: loopInputID, SourceHandle: "output", TargetHandle: "input"},
	}

	return &BuiltInFlow{
		Definition: &entity.FlowDefinition{
			Name:        "创建流程",
			Description: "通过对话创建新流程（系统内置）",
			Category:    "system",
			Config:      "{}",
			Icon:        "🛠️",
		},
		Nodes: nodes,
		Edges: edges,
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// topLevelFlow returns only top-level nodes and edges, filtering out children of container nodes (loop, etc.).
func topLevelFlow(nodes []*entity.FlowNode, edges []*entity.FlowEdge) ([]*entity.FlowNode, []*entity.FlowEdge) {
	topIDs := make(map[uint]bool)
	for _, n := range nodes {
		if n.GroupId == nil {
			topIDs[n.Id] = true
		}
	}
	mainNodes := make([]*entity.FlowNode, 0, len(topIDs))
	for _, n := range nodes {
		if n.GroupId == nil {
			mainNodes = append(mainNodes, n)
		}
	}
	mainEdges := make([]*entity.FlowEdge, 0, len(edges))
	for _, e := range edges {
		if topIDs[e.SourceNodeId] && topIDs[e.TargetNodeId] {
			mainEdges = append(mainEdges, e)
		}
	}
	return mainNodes, mainEdges
}

func (r *FlowRunner) Run() error {
	<-r.ctx.Done()
	return nil
}

func (r *FlowRunner) registerFunctions() {
	// LLM function
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

		// Non-streaming: for internal calls like for_each / iterator
		result, err := r.chatService.ChatWithHistoryWithContext(context.Background(), model, history, prompt, opts)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"output": result,
			"prompt": prompt,
		}, nil
	})

	// image_generation function
	r.functions.Register("image_generation", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		model, _ := args["model"].(string)
		if model == "" {
			model = r.chatService.GetDefaultPath()
		}
		prompt, _ := args["prompt"].(string)
		if prompt == "" {
			return nil, fmt.Errorf("image_generation: prompt is required")
		}
		count := 1
		if c, ok := args["count"].(float64); ok && c > 0 {
			count = int(c)
		}
		aspectRatio, _ := args["aspect_ratio"].(string)
		output, urls, err := r.genService.GenerateImage(context.Background(), model, prompt, count, aspectRatio)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"output": output,
			"urls":   urls,
			"count":  len(urls),
		}, nil
	})

	// audio_generation function
	r.functions.Register("audio_generation", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		model, _ := args["model"].(string)
		if model == "" {
			model = r.chatService.GetDefaultPath()
		}
		text, _ := args["text"].(string)
		if text == "" {
			return nil, fmt.Errorf("audio_generation: text is required")
		}
		voice, _ := args["voice"].(string)
		url, err := r.genService.GenerateAudio(context.Background(), model, text, voice)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"output": url,
			"url":    url,
		}, nil
	})

	// video_generation function
	r.functions.Register("video_generation", func(ctx *engine.ExecutionContext, name string, args map[string]any) (map[string]any, error) {
		model, _ := args["model"].(string)
		if model == "" {
			model = r.chatService.GetDefaultPath()
		}
		prompt, _ := args["prompt"].(string)
		if prompt == "" {
			return nil, fmt.Errorf("video_generation: prompt is required")
		}
		duration := 0
		if d, ok := args["duration"].(float64); ok {
			duration = int(d)
		}
		aspectRatio, _ := args["aspect_ratio"].(string)
		if aspectRatio == "" {
			aspectRatio, _ = args["aspect_ratio"].(string)
		}
		url, err := r.genService.GenerateVideo(context.Background(), model, prompt, duration, aspectRatio)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"output": url,
			"url":    url,
		}, nil
	})
}

// SetSendFunc sets the event sender (called by ChatRunner to inject WS capability)
func (r *FlowRunner) SetSendFunc(fn func(data []byte)) {
	r.sendFn = fn
}

// FlowStartOptions holds optional runtime overrides for a flow execution.
type FlowStartOptions struct {
	InitialInput    string
	FormValues      map[string]any
	ConfigOverrides map[string]string
}

func (r *FlowRunner) HandleFlowStart(flowId uint, executionId uint, sessionId uint, opts FlowStartOptions) (uint, error) {
	flowDef, err := r.flowModel.FindById(flowId)
	if err != nil {
		return 0, fmt.Errorf("flow not found: %w", err)
	}

	// Load flow content (nodes, edges, config) from disk via app store.
	content, err := r.appStore.LoadFlow(flowDef.Path)
	if err != nil {
		return 0, fmt.Errorf("failed to load flow from disk: %w", err)
	}

	// Hydrate on-disk fields onto the flow definition for the engine.
	flowDef.Config = content.Config
	flowDef.FormSchema = content.FormSchema
	flowDef.Settings = content.Settings

	return r.startExecution(flowId, flowDef, content.Nodes, content.Edges, executionId, sessionId, opts)
}

// HandleBuiltInFlowStart starts a system built-in flow by name.
func (r *FlowRunner) HandleBuiltInFlowStart(name string, executionId uint, sessionId uint, opts FlowStartOptions) (uint, error) {
	bf, ok := r.builtInFlows[name]
	if !ok {
		return 0, fmt.Errorf("built-in flow not found: %s", name)
	}
	return r.startExecution(0, bf.Definition, bf.Nodes, bf.Edges, executionId, sessionId, opts)
}

func (r *FlowRunner) startExecution(flowId uint, flowDef *entity.FlowDefinition, flowNodes []*entity.FlowNode, flowEdges []*entity.FlowEdge, executionId uint, sessionId uint, opts FlowStartOptions) (uint, error) {
	var exec *entity.FlowExecution
	var err error
	if executionId > 0 {
		exec, err = r.executionModel.FindById(executionId)
		if err != nil {
			return 0, fmt.Errorf("execution not found: %w", err)
		}
	} else {
		exec = &entity.FlowExecution{
			FlowId:    flowId,
			SessionId: sessionId,
			Status:    engine.ExecRunning,
			Context:   "{}",
		}
		if err := r.executionModel.Create(exec); err != nil {
			return 0, fmt.Errorf("failed to create execution: %w", err)
		}
		executionId = exec.Id
	}

	// Separate top-level nodes/edges from container children (e.g. loop bodies).
	mainNodes, mainEdges := topLevelFlow(flowNodes, flowEdges)

	eng := engine.NewEngine(r.registry, r)
	eng.SetTaskManager(r.taskMgr)
	eng.LoadFlow(mainNodes, mainEdges)

	startId, err := eng.FindStartNode()
	if err != nil {
		return 0, err
	}

	execCtx := engine.NewExecutionContext(flowId, executionId, sessionId, r)
	execCtx.Functions = r.functions
	execCtx.Cache = r.cacheMgr
	execCtx.Registry = r.registry
	execCtx.Nodes = flowNodes
	execCtx.Edges = flowEdges

	// Load config from FlowDefinition.Config (JSON string).
	if flowDef.Config != "" {
		var cfg map[string]string
		if err := json.Unmarshal([]byte(flowDef.Config), &cfg); err == nil {
			execCtx.SetConfig(cfg)
		}
	}
	// Apply runtime config overrides.
	if len(opts.ConfigOverrides) > 0 {
		for k, v := range opts.ConfigOverrides {
			execCtx.Config[k] = v
		}
		execCtx.SetConfig(execCtx.Config)
	}

	// Seed form values and initial input.
	if len(opts.FormValues) > 0 {
		execCtx.SetFormValues(opts.FormValues)
	}
	if opts.InitialInput != "" {
		execCtx.SeedInput(opts.InitialInput)
	}

	r.mu.Lock()
	r.running[executionId] = &runningExecution{
		engine:        eng,
		ctx:           execCtx,
		currentNodeId: startId,
		waitCh:        make(chan string, 1),
		doneCh:        make(chan struct{}),
	}
	r.mu.Unlock()

	exec.Status = engine.ExecRunning
	if err := r.executionModel.Update(exec); err != nil {
		log.Warn("Failed to update execution status", zap.Error(err))
	}

	snapshotStop := make(chan struct{})
	go r.snapshotLoop(executionId, exec, execCtx, snapshotStop)

	go func() {
		err := eng.Run(execCtx, startId)

		close(snapshotStop)
		r.snapshotContext(executionId, exec, execCtx)

		if err != nil {
			exec.Status = engine.ExecError
			exec.Context = fmt.Sprintf(`{"error":"%s"}`, err.Error())
		} else {
			exec.Status = engine.ExecCompleted
			ctxJSON, _ := json.Marshal(map[string]any{
				"data":         execCtx.Data,
				"node_outputs": execCtx.NodeOutputs,
			})
			exec.Context = string(ctxJSON)
		}
		// Persist status before removing from running map so status checks are consistent.
		r.executionModel.Update(exec)

		r.mu.Lock()
		if re, ok := r.running[executionId]; ok {
			close(re.doneCh)
		}
		delete(r.running, executionId)
		r.mu.Unlock()

		log.Info("Flow execution finished",
			zap.Uint("executionId", executionId),
			zap.Uint("flowId", flowId),
			zap.String("status", exec.Status))
	}()

	return executionId, nil
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

func (r *FlowRunner) GetExecutionStatus(executionId uint) (*entity.FlowExecution, error) {
	return r.executionModel.FindById(executionId)
}

// GetWaitingPrompt returns the current user-input prompt for a running execution, if any.
func (r *FlowRunner) GetWaitingPrompt(executionId uint) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if re, ok := r.running[executionId]; ok && re.ctx != nil {
		return re.ctx.WaitingPrompt
	}
	return ""
}

// snapshotContext serializes the current execution context into the FlowExecution row.
func (r *FlowRunner) snapshotContext(executionId uint, exec *entity.FlowExecution, ctx *engine.ExecutionContext) {
	nodeID, _, _ := ctx.GetCurrentNode()
	ctxJSON, err := json.Marshal(map[string]any{
		"data":          ctx.Data,
		"node_outputs":  ctx.NodeOutputs,
		"current_node":  nodeID,
		"waiting_prompt": ctx.GetWaitingPrompt(),
	})
	if err != nil {
		log.Warn("Failed to marshal snapshot", zap.Uint("executionId", executionId), zap.Error(err))
		return
	}
	exec.Context = string(ctxJSON)
	exec.CurrentNodeId = &nodeID
	if err := r.executionModel.Update(exec); err != nil {
		log.Warn("Failed to persist snapshot", zap.Uint("executionId", executionId), zap.Error(err))
	}
}

// snapshotLoop periodically snapshots the execution context until stop is closed.
func (r *FlowRunner) snapshotLoop(executionId uint, exec *entity.FlowExecution, ctx *engine.ExecutionContext, stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if ctx.IsAborted() {
				return
			}
			r.snapshotContext(executionId, exec, ctx)
		}
	}
}

// GetWaitChannels returns the wait/done channels for a running execution.
// Returns nil channels if the execution is not running.
func (r *FlowRunner) GetWaitChannels(executionId uint) (chan string, chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if re, ok := r.running[executionId]; ok {
		return re.waitCh, re.doneCh
	}
	return nil, nil
}

// Emit implements the engine.EventEmitter interface
func (r *FlowRunner) Emit(event engine.FlowEvent) {
	r.mu.Lock()
	re, ok := r.running[event.ExecutionId]
	sessionId := uint(0)
	if ok && re.ctx != nil {
		sessionId = re.ctx.SessionId
		// Forward waiting prompts to any blocking tool call.
		if event.Type == engine.EventWaitingUser {
			select {
			case re.waitCh <- event.Message:
			default:
			}
		}
	}
	r.mu.Unlock()

	if r.sendFn == nil {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		log.Error("Failed to serialize flow event", zap.Error(err))
		return
	}
	// Enrich event with session_id so desktop IPC can route it.
	if sessionId > 0 {
		var eventMap map[string]any
		if err := json.Unmarshal(data, &eventMap); err == nil {
			eventMap["session_id"] = sessionId
			data, _ = json.Marshal(eventMap)
		}
	}
	r.sendFn(data)
}
