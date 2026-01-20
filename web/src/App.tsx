import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import LoginPage from './pages/LoginPage'
import ChatPage from './pages/ChatPage'

const App = () => {
  const token = useAuthStore((state) => state.token)

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/chat" element={token ? <ChatPage /> : <Navigate to="/login" />} />
      <Route path="*" element={<Navigate to={token ? '/chat' : '/login'} />} />
    </Routes>
  )
}

export default App
