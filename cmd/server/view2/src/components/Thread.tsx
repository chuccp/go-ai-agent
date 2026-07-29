import {
  ThreadPrimitive,
  MessagePrimitive,
  ComposerPrimitive,
  useThread,
  useComposerRuntime,
} from '@assistant-ui/react'

export function Thread() {
  return (
    <ThreadPrimitive.Root style={{
      display: 'flex', flexDirection: 'column', height: '100%', background: '#fff',
    }}>
      {/* ── Header ── */}
      <div style={{
        padding: '12px 24px', borderBottom: '1px solid #e8eaed',
        display: 'flex', alignItems: 'center', gap: 8,
        background: '#fafbfc', flexShrink: 0,
      }}>
        <div style={{
          width: 28, height: 28, borderRadius: '50%',
          background: 'linear-gradient(135deg, #4285f4, #34a853)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontSize: 14, fontWeight: 700,
        }}>AI</div>
        <span style={{ fontWeight: 600, fontSize: 15, color: '#202124' }}>Agent Debug Chat</span>
        <span style={{ fontSize: 11, color: '#80868b', marginLeft: 4 }}>
          go-agent-sdk · {location.hostname}:19009
        </span>
      </div>

      {/* ── Messages ── */}
      <ThreadPrimitive.Viewport style={{ flex: 1, overflowY: 'auto', padding: '24px 0' }}>
        <div style={{ maxWidth: 820, margin: '0 auto', padding: '0 24px' }}>
          <ThreadPrimitive.Empty>
            <div style={{
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              justifyContent: 'center', minHeight: '40vh', gap: 12,
            }}>
              <div style={{ fontSize: 32, fontWeight: 700, color: '#1a73e8' }}>🐛 Agent Debug</div>
              <div style={{ fontSize: 14, color: '#5f6368' }}>
                Send a message to test the agent — WebSocket at /ws/chat
              </div>
              <div style={{ fontSize: 12, color: '#80868b', fontFamily: 'monospace' }}>
                WS protocol: {'{"type":"chat","message":"..."}'}<br />
                Response: {'{"type":"chunk","content":"..."} → {"type":"done"}'}
              </div>
            </div>
          </ThreadPrimitive.Empty>

          <ThreadPrimitive.Messages
            components={{
              UserMessage,
              AssistantMessage,
            }}
          />
        </div>
      </ThreadPrimitive.Viewport>

      {/* ── Composer ── */}
      <div style={{ padding: '0 24px 24px', flexShrink: 0 }}>
        <div style={{ maxWidth: 820, margin: '0 auto' }}>
          <Composer />
        </div>
      </div>
    </ThreadPrimitive.Root>
  )
}

// ── User Message Bubble ──

function UserMessage() {
  return (
    <MessagePrimitive.Root style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 24 }}>
      <div style={{ maxWidth: '70%', display: 'flex', alignItems: 'flex-start', gap: 12 }}>
        <div style={{
          padding: '12px 18px', borderRadius: '20px 20px 4px 20px',
          background: '#1a73e8', color: '#fff', fontSize: 14, lineHeight: 1.7,
          wordBreak: 'break-word',
        }}>
          <MessagePrimitive.Content />
        </div>
        <div style={{
          width: 34, height: 34, borderRadius: '50%', background: '#e8f0fe',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#1a73e8', fontSize: 13, fontWeight: 600, flexShrink: 0,
        }}>U</div>
      </div>
    </MessagePrimitive.Root>
  )
}

// ── Assistant Message (agent-style, full width, no bubble) ──

function AssistantMessage() {
  return (
    <MessagePrimitive.Root style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 32 }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, width: '100%' }}>
        <div style={{
          width: 30, height: 30, borderRadius: '50%',
          background: 'linear-gradient(135deg, #4285f4, #34a853)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontSize: 14, fontWeight: 700, flexShrink: 0,
          marginTop: 2,
        }}>AI</div>
        <div style={{
          fontSize: 15, lineHeight: 1.75, color: '#1f2937',
          wordBreak: 'break-word', flex: 1, minWidth: 0,
        }}>
          <MessagePrimitive.Content />
        </div>
      </div>
    </MessagePrimitive.Root>
  )
}

// ── Composer (input bar) ──

function Composer() {
  const isRunning = useThread((state) => state.isRunning)
  const composerRuntime = useComposerRuntime()

  return (
    <div>
      <ComposerPrimitive.Root style={{
        display: 'flex', alignItems: 'flex-end', borderRadius: 24,
        border: '1px solid #dadce0', background: '#fff',
        padding: '8px 8px 8px 16px',
        transition: 'border-color 0.15s, box-shadow 0.15s',
        boxShadow: '0 2px 6px rgba(0,0,0,0.06)',
      }}>
        <ComposerPrimitive.Input
          style={{
            flex: 1, border: 'none', outline: 'none', fontSize: 15,
            lineHeight: '24px', resize: 'none', fontFamily: 'inherit',
            padding: '6px 8px', background: 'transparent', color: '#202124',
          }}
          placeholder="Type a message... (Enter to send)"
          rows={1}
        />

        {isRunning ? (
          <button
            onClick={() => composerRuntime.cancel()}
            style={{
              width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: '#ea4335', color: '#fff', cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              flexShrink: 0,
            }}
            title="Stop"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <rect x="6" y="6" width="12" height="12" rx="2" />
            </svg>
          </button>
        ) : (
          <ComposerPrimitive.Send
            style={{
              width: 36, height: 36, borderRadius: '50%', border: 'none',
              background: '#1a73e8', color: '#fff', cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M22 2L11 13" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <path d="M22 2L15 22L11 13L2 9L22 2Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </ComposerPrimitive.Send>
        )}
      </ComposerPrimitive.Root>

      <div style={{
        display: 'flex', alignItems: 'center', marginTop: 6, paddingLeft: 4,
      }}>
        <span style={{ fontSize: 11, color: '#80868b' }}>Enter 发送 · 用于调试 agent 的纯聊天界面</span>
        <span style={{ marginLeft: 'auto', fontSize: 10, color: '#dadce0', fontFamily: 'monospace' }}>
          /ws/chat
        </span>
      </div>
    </div>
  )
}
