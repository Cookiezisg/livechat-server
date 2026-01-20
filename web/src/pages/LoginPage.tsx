import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

const LoginPage = () => {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const { login, register, loading, error } = useAuthStore()
  const navigate = useNavigate()

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (mode === 'login') {
      await login(email, password)
    } else {
      await register(email, username, password)
    }
    if (useAuthStore.getState().token) {
      navigate('/chat')
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <div className="auth-header">
          <span className="badge">Livechat Studio</span>
          <h1>{mode === 'login' ? 'Welcome back.' : 'Create your space.'}</h1>
          <p>
            {mode === 'login'
              ? 'Sign in to pick up the conversation across rooms and DMs.'
              : 'Spin up a profile to join the new studio-grade chat.'}
          </p>
        </div>
        <form onSubmit={handleSubmit} className="auth-form">
          <label>
            Email
            <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
          </label>
          {mode === 'register' && (
            <label>
              Username
              <input value={username} onChange={(e) => setUsername(e.target.value)} type="text" required />
            </label>
          )}
          <label>
            Password
            <input value={password} onChange={(e) => setPassword(e.target.value)} type="password" required />
          </label>
          {error && <div className="form-error">{error}</div>}
          <button type="submit" disabled={loading}>
            {loading ? 'Loading...' : mode === 'login' ? 'Enter Chat' : 'Create Account'}
          </button>
        </form>
        <div className="auth-footer">
          <button
            type="button"
            className="link"
            onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
          >
            {mode === 'login' ? 'Need an account? Register' : 'Already have an account? Sign in'}
          </button>
        </div>
      </div>
      <div className="auth-panel">
        <div className="panel-copy">
          <h2>Rooms, DMs, and live presence in one canvas.</h2>
          <p>
            Built with WebSocket streams, Postgres persistence, and a polished frontend stack to keep
            everything responsive.
          </p>
        </div>
        <div className="stat-grid">
          <div>
            <span>Realtime</span>
            <strong>WebSocket core</strong>
          </div>
          <div>
            <span>Persistence</span>
            <strong>Postgres + Redis</strong>
          </div>
          <div>
            <span>Secure</span>
            <strong>JWT sessions</strong>
          </div>
        </div>
      </div>
    </div>
  )
}

export default LoginPage
