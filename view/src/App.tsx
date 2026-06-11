import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useSetupStore } from '@/stores/setupStore'
import ChatHome from '@/pages/ChatHome'
import FlowDesigner from '@/pages/FlowDesigner'
import ModelManager from '@/pages/ModelManager'
import SetupWizard from '@/pages/SetupWizard'

export default function App() {
  const { initialized, checkSetup } = useSetupStore()
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    checkSetup().finally(() => setChecking(false))
  }, [checkSetup])

  if (checking || initialized === null) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
        <div className="spinner" />
      </div>
    )
  }

  return (
    <HashRouter>
      <Routes>
        <Route path="/setup" element={<SetupWizard />} />
        {!initialized && <Route path="*" element={<Navigate to="/setup" replace />} />}
        {initialized && (
          <>
            <Route path="/" element={<ChatHome />} />
            <Route path="/designer" element={<FlowDesigner />} />
            <Route path="/designer/:id" element={<FlowDesigner />} />
            <Route path="/models" element={<ModelManager />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </>
        )}
      </Routes>
    </HashRouter>
  )
}
