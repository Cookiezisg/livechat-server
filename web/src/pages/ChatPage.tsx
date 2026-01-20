import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '../store/auth'
import { useChatStore } from '../store/chat'
import { Message, Room, User } from '../lib/api'

const ChatPage = () => {
  const { token, user, logout, hydrate } = useAuthStore()
  const {
    rooms,
    activeRoomId,
    activeDMUserId,
    messages,
    presence,
    typing,
    loadRooms,
    setActiveRoom,
    setActiveDM,
    connect,
    sendMessage,
    sendTyping,
    loadHistory,
    searchUsers,
  } = useChatStore()
  const [composer, setComposer] = useState('')
  const [search, setSearch] = useState('')
  const [dmResults, setDmResults] = useState<User[]>([])

  useEffect(() => {
    hydrate()
  }, [hydrate])

  useEffect(() => {
    if (!token || !user) return
    loadRooms(token)
    connect(token, user.id)
  }, [token, user, loadRooms, connect])

  useEffect(() => {
    if (!token) return
    if (activeRoomId) {
      loadHistory(token, activeRoomId)
    } else if (activeDMUserId) {
      loadHistory(token, undefined, activeDMUserId)
    }
  }, [token, activeRoomId, activeDMUserId, loadHistory])

  useEffect(() => {
    const timeout = setTimeout(async () => {
      if (!token || search.trim().length < 2) {
        setDmResults([])
        return
      }
      const results = await searchUsers(token, search.trim())
      setDmResults(results)
    }, 350)
    return () => clearTimeout(timeout)
  }, [search, token, searchUsers])

  const activeKey = activeRoomId ?? (activeDMUserId ? `dm:${activeDMUserId}` : '')
  const thread = messages[activeKey] || []
  const activeTyping = typing[activeKey] || []

  const handleSend = () => {
    if (!composer.trim()) return
    sendMessage({
      roomId: activeRoomId || undefined,
      recipientId: activeDMUserId || undefined,
      body: composer.trim(),
    })
    setComposer('')
  }

  const headerTitle = useMemo(() => {
    if (activeRoomId) {
      const room = rooms.find((r) => r.id === activeRoomId)
      return room ? `#${room.name}` : 'Room'
    }
    if (activeDMUserId) {
      const dmUser = dmResults.find((u) => u.id === activeDMUserId)
      return dmUser ? dmUser.username : 'Direct Message'
    }
    return 'Choose a room'
  }, [activeRoomId, activeDMUserId, rooms, dmResults])

  return (
    <div className="chat-shell">
      <aside className="sidebar">
        <div className="brand">
          <div>
            <h2>Livechat Studio</h2>
            <p>{user?.username}</p>
          </div>
          <button onClick={logout} className="ghost">
            Logout
          </button>
        </div>
        <section>
          <h3>Rooms</h3>
          <div className="list">
            {rooms.map((room) => (
              <button
                key={room.id}
                className={activeRoomId === room.id ? 'active' : ''}
                onClick={() => setActiveRoom(room.id)}
              >
                <span># {room.name}</span>
              </button>
            ))}
          </div>
        </section>
        <section>
          <h3>Directs</h3>
          <input
            className="search"
            placeholder="Search people"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <div className="list">
            {dmResults.map((person) => (
              <button
                key={person.id}
                className={activeDMUserId === person.id ? 'active' : ''}
                onClick={() => setActiveDM(person.id)}
              >
                <span>{person.username}</span>
                <span className={presence[person.id]?.online ? 'dot online' : 'dot'} />
              </button>
            ))}
          </div>
        </section>
      </aside>
      <main className="chat-main">
        <header className="chat-header">
          <div>
            <h1>{headerTitle}</h1>
            <p>{activeRoomId ? 'Live room feed' : activeDMUserId ? 'Private chat' : 'Pick a thread'}</p>
          </div>
        </header>
        <div className="chat-thread">
          {thread.length === 0 && <div className="empty">No messages yet. Start the conversation.</div>}
          {thread.map((message) => (
            <MessageBubble key={message.id} message={message} selfId={user?.id || ''} />
          ))}
        </div>
        {activeTyping.length > 0 && (
          <div className="typing">{activeTyping.length} typing...</div>
        )}
        <div className="composer">
          <input
            value={composer}
            onChange={(e) => {
              setComposer(e.target.value)
              sendTyping({
                roomId: activeRoomId || undefined,
                recipientId: activeDMUserId || undefined,
                isTyping: e.target.value.length > 0,
              })
            }}
            placeholder="Write a message"
          />
          <button onClick={handleSend}>Send</button>
        </div>
      </main>
    </div>
  )
}

const MessageBubble = ({ message, selfId }: { message: Message; selfId: string }) => {
  const isSelf = message.senderId === selfId
  return (
    <div className={`message ${isSelf ? 'self' : ''}`}>
      <div className="meta">
        <span>{message.senderName}</span>
        <time>{new Date(message.createdAt).toLocaleTimeString()}</time>
      </div>
      <div className="bubble">
        <p>{message.body}</p>
        {message.attachmentUrl && (
          <a href={message.attachmentUrl} target="_blank" rel="noreferrer">
            Attachment
          </a>
        )}
      </div>
    </div>
  )
}

export default ChatPage
