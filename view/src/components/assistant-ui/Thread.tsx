import {
  ThreadPrimitive,
  MessagePrimitive,
  ComposerPrimitive,
  useThread,
  useComposerRuntime,
} from '@assistant-ui/react'
import { useTranslation } from 'react-i18next'

interface ThreadProps {
  models?: { id: string; name: string }[]
  selectedModelId?: string
  thinkLevel?: string
  onModelChange?: (id: string) => void
  onThinkChange?: (level: string) => void
}

export function Thread({ models, selectedModelId, thinkLevel, onModelChange, onThinkChange }: ThreadProps) {
  const { t } = useTranslation()

  return (
    <ThreadPrimitive.Root style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      background: '#fff',
    }}>
      <ThreadPrimitive.Viewport style={{
        flex: 1,
        overflowY: 'auto',
        padding: '24px 0',
      }}>
        <div style={{ maxWidth: 820, margin: '0 auto', padding: '0 24px' }}>
          <ThreadPrimitive.Empty>
            <div style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: '50vh',
              gap: 16,
            }}>
              <div style={{ fontSize: 40, fontWeight: 600, color: '#1a73e8' }}>Go AI Agent</div>
              <div style={{ fontSize: 16, color: '#5f6368' }}>{t('chat.selectModelHint')}</div>
            </div>
          </ThreadPrimitive.Empty>

          <ThreadPrimitive.Messages
            components={{
              UserMessage: UserMessage,
              AssistantMessage: AssistantMessage,
            }}
          />
        </div>
      </ThreadPrimitive.Viewport>

      <div style={{
        padding: '0 24px 24px',
        flexShrink: 0,
      }}>
        <div style={{ maxWidth: 820, margin: '0 auto' }}>
          <Composer models={models} selectedModelId={selectedModelId} thinkLevel={thinkLevel} onModelChange={onModelChange} onThinkChange={onThinkChange} />
        </div>
      </div>
    </ThreadPrimitive.Root>
  )
}

function UserMessage() {
  return (
    <MessagePrimitive.Root style={{
      display: 'flex',
      justifyContent: 'flex-end',
      marginBottom: 24,
    }}>
      <div style={{ maxWidth: '70%', display: 'flex', alignItems: 'flex-start', gap: 12 }}>
        <div style={{
          padding: '12px 18px',
          borderRadius: '20px 20px 4px 20px',
          background: '#1a73e8',
          color: '#fff',
          fontSize: 14,
          lineHeight: 1.7,
          wordBreak: 'break-word',
        }}>
          <MessagePrimitive.Content />
        </div>
        <div style={{
          width: 36, height: 36, borderRadius: '50%',
          background: '#e8f0fe',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#1a73e8', fontSize: 14, fontWeight: 600,
          flexShrink: 0,
        }}>U</div>
      </div>
    </MessagePrimitive.Root>
  )
}

function AssistantMessage() {
  return (
    <MessagePrimitive.Root style={{
      display: 'flex',
      justifyContent: 'flex-start',
      marginBottom: 24,
    }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, maxWidth: '85%' }}>
        <div style={{
          width: 36, height: 36, borderRadius: '50%',
          background: 'linear-gradient(135deg, #4285f4, #34a853)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontSize: 16, fontWeight: 700,
          flexShrink: 0,
        }}>AI</div>
        <div style={{
          padding: '12px 16px',
          borderRadius: '20px 20px 20px 4px',
          background: '#f8f9fa',
          border: '1px solid #e8eaed',
          fontSize: 14,
          lineHeight: 1.7,
          color: '#3c4043',
          wordBreak: 'break-word',
        }}>
          <MessagePrimitive.Content />
        </div>
      </div>
    </MessagePrimitive.Root>
  )
}

function Composer({ models, selectedModelId, thinkLevel, onModelChange, onThinkChange }: ThreadProps) {
  const { t } = useTranslation()
  const isRunning = useThread((state) => state.isRunning)
  const composerRuntime = useComposerRuntime()

  const handleStop = () => {
    composerRuntime.cancel()
  }

  return (
    <div>
      <ComposerPrimitive.Root style={{
        display: 'flex',
        alignItems: 'flex-end',
        borderRadius: 24,
        border: '1px solid #dadce0',
        background: '#fff',
        padding: '8px 8px 8px 16px',
        transition: 'border-color 0.15s, box-shadow 0.15s',
        boxShadow: '0 2px 6px rgba(0,0,0,0.06)',
      }}>
        {/* Attach button */}
        <button
          style={{
            width: 32, height: 32, borderRadius: '50%',
            border: 'none', background: 'transparent',
            cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
            color: '#5f6368', flexShrink: 0,
          }}
          title={t('chat.uploadFile')}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
            <path d="M12 5v14M5 12h14"/>
          </svg>
        </button>

        <ComposerPrimitive.Input
          style={{
            flex: 1,
            border: 'none',
            outline: 'none',
            fontSize: 15,
            lineHeight: '24px',
            resize: 'none',
            fontFamily: 'inherit',
            padding: '6px 8px',
            background: 'transparent',
            color: '#202124',
          }}
          placeholder={t('chat.inputPlaceholder')}
          rows={1}
        />

        {/* Send / Stop button */}
        {isRunning ? (
          <button
            onClick={handleStop}
            style={{
              width: 36, height: 36,
              borderRadius: '50%',
              border: 'none',
              background: '#ea4335',
              color: '#fff',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
              transition: 'background 0.15s',
            }}
            title={t('chat.stop')}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <rect x="6" y="6" width="12" height="12" rx="2" />
            </svg>
          </button>
        ) : (
          <ComposerPrimitive.Send
            style={{
              width: 36, height: 36,
              borderRadius: '50%',
              border: 'none',
              background: '#1a73e8',
              color: '#fff',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
              transition: 'background 0.15s',
            }}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M22 2L11 13" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <path d="M22 2L15 22L11 13L2 9L22 2Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </ComposerPrimitive.Send>
        )}
      </ComposerPrimitive.Root>

      {/* Bottom bar: hint + selectors */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        marginTop: 8,
        paddingLeft: 4,
      }}>
        <span style={{ fontSize: 11, color: '#80868b' }}>{t('chat.sendHint')}</span>
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 6 }}>
          {models && models.length > 0 && (
            <select
              value={selectedModelId}
              onChange={e => onModelChange?.(e.target.value)}
              style={{
                padding: '4px 8px', borderRadius: 12,
                border: '1px solid #dadce0', fontSize: 12,
                outline: 'none', background: '#fff', color: '#5f6368',
                cursor: 'pointer', maxWidth: 180,
              }}
            >
              {models.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
            </select>
          )}
          <select
            value={thinkLevel}
            onChange={e => onThinkChange?.(e.target.value)}
            style={{
              padding: '4px 8px', borderRadius: 12,
              border: '1px solid #dadce0', fontSize: 12,
              outline: 'none', background: '#fff', color: '#5f6368',
              cursor: 'pointer', maxWidth: 100,
            }}
          >
            {['off', 'low', 'medium', 'high', 'max'].map(lv => (
              <option key={lv} value={lv}>{t(`think.${lv}`)}</option>
            ))}
          </select>
        </div>
      </div>
    </div>
  )
}
