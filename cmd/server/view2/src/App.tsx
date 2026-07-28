import { ChatRuntimeProvider } from './components/ChatRuntimeProvider'
import { Thread } from './components/Thread'

export default function App() {
  return (
    <ChatRuntimeProvider>
      <Thread />
    </ChatRuntimeProvider>
  )
}
