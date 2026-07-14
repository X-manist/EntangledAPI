/**
 * Chat Store
 * 后端优先持久化；localStorage 仅作为离线/异常兜底缓存。
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth'
import { chatAPI, type ChatConversationDTO, type ChatMessageDTO, type ChatRole } from '@/api/chat'

const CONVERSATIONS_KEY = 'sub2api_conversations'
const ACTIVE_CONVERSATION_KEY = 'sub2api_active_conversation'
const DEFAULT_MODEL = 'claude-3-5-sonnet-20241022'

export interface Message {
  id: string
  role: ChatRole
  content: string
  timestamp: number
  tokens?: number
  cost?: number
  model?: string
}

export interface Conversation {
  id: string
  title: string
  messages: Message[]
  createdAt: number
  updatedAt: number
  metadata?: {
    model?: string
    systemPrompt?: string
    backendSynced?: boolean
    [key: string]: unknown
  }
}

function toTimestamp(value: string | number | undefined): number {
  if (typeof value === 'number') return value
  if (!value) return Date.now()
  const parsed = new Date(value).getTime()
  return Number.isFinite(parsed) ? parsed : Date.now()
}

function fromBackendMessage(input: ChatMessageDTO): Message {
  return {
    id: String(input.id),
    role: input.role,
    content: input.content,
    timestamp: toTimestamp(input.created_at),
    tokens: input.tokens_used || undefined,
    cost: input.cost || undefined,
    model: input.model
  }
}

function fromBackendConversation(input: ChatConversationDTO): Conversation {
  const metadata = (input.metadata || {}) as Conversation['metadata']
  return {
    id: String(input.id),
    title: input.title || '新对话',
    messages: (input.messages || []).map(fromBackendMessage),
    createdAt: toTimestamp(input.created_at),
    updatedAt: toTimestamp(input.updated_at),
    metadata: {
      ...metadata,
      backendSynced: true
    }
  }
}

function createLocalConversation(title?: string): Conversation {
  const now = Date.now()
  return {
    id: `local_${now}_${Math.random().toString(36).slice(2, 11)}`,
    title: title || '新对话',
    messages: [],
    createdAt: now,
    updatedAt: now,
    metadata: { backendSynced: false }
  }
}

function createLocalMessage(message: Omit<Message, 'id' | 'timestamp'>): Message {
  return {
    id: `local_msg_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`,
    timestamp: Date.now(),
    ...message
  }
}

export const useChatStore = defineStore('chat', () => {
  const authStore = useAuthStore()

  const conversations = ref<Conversation[]>([])
  const activeConversationId = ref<string | null>(null)
  const isLoading = ref(false)
  const isStreaming = ref(false)
  const streamingMessageId = ref<string | null>(null)
  const backendAvailable = ref(true)

  const activeConversation = computed(() => {
    if (!activeConversationId.value) return null
    return conversations.value.find(c => c.id === activeConversationId.value) || null
  })

  const sortedConversations = computed(() => {
    return [...conversations.value].sort((a, b) => b.updatedAt - a.updatedAt)
  })

  function saveToStorage() {
    try {
      localStorage.setItem(CONVERSATIONS_KEY, JSON.stringify(conversations.value))
      if (activeConversationId.value) {
        localStorage.setItem(ACTIVE_CONVERSATION_KEY, activeConversationId.value)
      } else {
        localStorage.removeItem(ACTIVE_CONVERSATION_KEY)
      }
    } catch (error) {
      console.error('Failed to save conversations to localStorage:', error)
    }
  }

  function loadFromStorage() {
    try {
      const stored = localStorage.getItem(CONVERSATIONS_KEY)
      conversations.value = stored ? JSON.parse(stored) : []
      const activeId = localStorage.getItem(ACTIVE_CONVERSATION_KEY)
      activeConversationId.value = activeId && conversations.value.some(c => c.id === activeId) ? activeId : conversations.value[0]?.id || null
    } catch (error) {
      console.error('Failed to load conversations from localStorage:', error)
      conversations.value = []
      activeConversationId.value = null
    }
  }

  async function loadConversations() {
    if (!authStore.isAuthenticated) {
      loadFromStorage()
      return
    }

    isLoading.value = true
    try {
      const result = await chatAPI.listConversations()
      const summaries = result.items.map(fromBackendConversation)
      conversations.value = summaries
      backendAvailable.value = true

      const storedActiveId = localStorage.getItem(ACTIVE_CONVERSATION_KEY)
      activeConversationId.value = storedActiveId && summaries.some(c => c.id === storedActiveId) ? storedActiveId : summaries[0]?.id || null
      saveToStorage()
    } catch (error) {
      backendAvailable.value = false
      console.warn('Backend chat storage unavailable, falling back to localStorage:', error)
      loadFromStorage()
    } finally {
      isLoading.value = false
    }
  }

  async function ensureConversationMessages(id: string) {
    const conversation = conversations.value.find(c => c.id === id)
    if (!conversation || conversation.id.startsWith('local_') || conversation.messages.length > 0) return

    try {
      const detail = await chatAPI.getConversation(id)
      const hydrated = fromBackendConversation(detail)
      const index = conversations.value.findIndex(c => c.id === id)
      if (index !== -1) {
        conversations.value[index] = hydrated
        saveToStorage()
      }
    } catch (error) {
      console.warn('Failed to hydrate conversation messages:', error)
    }
  }

  async function createConversation(title?: string): Promise<Conversation> {
    if (authStore.isAuthenticated && backendAvailable.value) {
      try {
        const created = await chatAPI.createConversation({ title: title || '新对话' })
        const conversation = fromBackendConversation(created)
        conversations.value.unshift(conversation)
        activeConversationId.value = conversation.id
        saveToStorage()
        return conversation
      } catch (error) {
        backendAvailable.value = false
        console.warn('Create backend conversation failed, using local fallback:', error)
      }
    }

    const conversation = createLocalConversation(title)
    conversations.value.unshift(conversation)
    activeConversationId.value = conversation.id
    saveToStorage()
    return conversation
  }

  async function selectConversation(id: string) {
    const conversation = conversations.value.find(c => c.id === id)
    if (!conversation) return
    activeConversationId.value = id
    localStorage.setItem(ACTIVE_CONVERSATION_KEY, id)
    await ensureConversationMessages(id)
  }

  async function deleteConversation(id: string) {
    const conversation = conversations.value.find(c => c.id === id)
    if (!conversation) return

    if (!id.startsWith('local_') && backendAvailable.value) {
      try {
        await chatAPI.deleteConversation(id)
      } catch (error) {
        console.warn('Delete backend conversation failed, removing locally only:', error)
      }
    }

    const index = conversations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      conversations.value.splice(index, 1)
      if (activeConversationId.value === id) {
        activeConversationId.value = conversations.value[0]?.id || null
      }
      saveToStorage()
    }
  }

  async function updateConversationTitle(id: string, title: string) {
    const conversation = conversations.value.find(c => c.id === id)
    if (!conversation) return

    const previous = conversation.title
    conversation.title = title
    conversation.updatedAt = Date.now()
    saveToStorage()

    if (!id.startsWith('local_') && backendAvailable.value) {
      try {
        const updated = await chatAPI.updateConversation(id, { title })
        conversation.title = updated.title || title
        conversation.updatedAt = toTimestamp(updated.updated_at)
        conversation.metadata = {
          ...(updated.metadata as Conversation['metadata']),
          backendSynced: true
        }
      } catch (error) {
        conversation.title = previous
        saveToStorage()
        throw error
      }
    }
  }

  function addMessage(conversationId: string, message: Omit<Message, 'id' | 'timestamp'>) {
    const conversation = conversations.value.find(c => c.id === conversationId)
    if (!conversation) return null

    const newMessage = createLocalMessage(message)
    conversation.messages.push(newMessage)
    conversation.updatedAt = Date.now()

    if (conversation.title === '新对话' && message.role === 'user') {
      const preview = message.content.slice(0, 30)
      conversation.title = preview.length < message.content.length ? `${preview}...` : preview
    }

    saveToStorage()
    return newMessage
  }

  function updateMessage(conversationId: string, messageId: string, updates: Partial<Message>) {
    const conversation = conversations.value.find(c => c.id === conversationId)
    if (!conversation) return

    const message = conversation.messages.find(m => m.id === messageId)
    if (message) {
      Object.assign(message, updates)
      conversation.updatedAt = Date.now()
      saveToStorage()
    }
  }

  async function persistMessage(conversationId: string, message: Message) {
    if (conversationId.startsWith('local_') || !backendAvailable.value) return message

    const saved = await chatAPI.createMessage(conversationId, {
      role: message.role,
      content: message.content,
      tokens_used: message.tokens,
      cost: message.cost,
      model: message.model,
      metadata: {}
    })
    return fromBackendMessage(saved)
  }

  async function sendMessage(conversationId: string, content: string): Promise<void> {
    const conversation = conversations.value.find(c => c.id === conversationId)
    if (!conversation) throw new Error('Conversation not found')

    let userMessage = addMessage(conversationId, { role: 'user', content })
    if (!userMessage) throw new Error('Failed to create user message')

    try {
      const persisted = await persistMessage(conversationId, userMessage)
      if (persisted.id !== userMessage.id) {
        updateMessage(conversationId, userMessage.id, persisted)
        userMessage = persisted
      }
    } catch (error) {
      console.warn('Persist user message failed, continuing with local cache:', error)
    }

    const recentMessages = conversation.messages.slice(-10).map(m => ({
      role: m.role,
      content: m.content
    }))

    isStreaming.value = true
    const assistantMessage = addMessage(conversationId, {
      role: 'assistant',
      content: '',
      model: conversation.metadata?.model || DEFAULT_MODEL
    })

    if (!assistantMessage) {
      isStreaming.value = false
      throw new Error('Failed to create assistant message')
    }

    streamingMessageId.value = assistantMessage.id

    try {
      const apiKey = authStore.token
      if (!apiKey) throw new Error('Not authenticated')

      const response = await fetch('/v1/messages', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey}`,
          'Content-Type': 'application/json',
          'anthropic-version': '2023-06-01'
        },
        body: JSON.stringify({
          model: conversation.metadata?.model || DEFAULT_MODEL,
          messages: recentMessages,
          max_tokens: 4096,
          stream: true
        })
      })

      if (!response.ok) throw new Error(`API error: ${response.status}`)

      const reader = response.body?.getReader()
      if (!reader) throw new Error('No response body')

      const decoder = new TextDecoder()
      let buffer = ''
      let fullContent = ''
      let tokens: number | undefined
      let cost: number | undefined

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const data = line.slice(6).trim()
          if (data === '[DONE]') continue

          try {
            const parsed = JSON.parse(data)
            const delta =
              parsed.type === 'content_block_delta' && parsed.delta?.text
                ? parsed.delta.text
                : parsed.delta?.content || parsed.choices?.[0]?.delta?.content || ''

            if (delta) {
              fullContent += delta
              updateMessage(conversationId, assistantMessage.id, { content: fullContent })
            }

            if (parsed.usage) {
              tokens = parsed.usage.total_tokens || parsed.usage.output_tokens || tokens
              cost = parsed.usage.cost || cost
              updateMessage(conversationId, assistantMessage.id, { tokens, cost })
            }
          } catch (error) {
            console.warn('Failed to parse SSE data:', error)
          }
        }
      }

      if (!fullContent) throw new Error('No content received from API')

      const finalAssistant = {
        ...assistantMessage,
        content: fullContent,
        tokens,
        cost
      }
      try {
        const persisted = await persistMessage(conversationId, finalAssistant)
        if (persisted.id !== assistantMessage.id) {
          updateMessage(conversationId, assistantMessage.id, persisted)
        }
      } catch (error) {
        console.warn('Persist assistant message failed, kept local cache:', error)
      }
    } catch (error: any) {
      updateMessage(conversationId, assistantMessage.id, {
        content: `错误: ${error.message || '发送失败，请重试'}`
      })
      throw error
    } finally {
      isStreaming.value = false
      streamingMessageId.value = null
    }
  }

  async function clearAll() {
    const ids = conversations.value.map(c => c.id)
    conversations.value = []
    activeConversationId.value = null
    saveToStorage()
    localStorage.removeItem(CONVERSATIONS_KEY)
    localStorage.removeItem(ACTIVE_CONVERSATION_KEY)

    if (backendAvailable.value) {
      await Promise.allSettled(ids.filter(id => !id.startsWith('local_')).map(id => chatAPI.deleteConversation(id)))
    }
  }

  function exportConversation(id: string): string {
    const conversation = conversations.value.find(c => c.id === id)
    return conversation ? JSON.stringify(conversation, null, 2) : ''
  }

  function importConversation(jsonStr: string): boolean {
    try {
      const conversation = JSON.parse(jsonStr) as Conversation
      if (!conversation.id || !conversation.messages) return false

      const existing = conversations.value.findIndex(c => c.id === conversation.id)
      if (existing !== -1) {
        conversations.value[existing] = conversation
      } else {
        conversations.value.unshift(conversation)
      }
      saveToStorage()
      return true
    } catch (error) {
      console.error('Import failed:', error)
      return false
    }
  }

  loadFromStorage()

  return {
    conversations: computed(() => conversations.value),
    activeConversationId: computed(() => activeConversationId.value),
    activeConversation,
    sortedConversations,
    isLoading: computed(() => isLoading.value),
    isStreaming: computed(() => isStreaming.value),
    streamingMessageId: computed(() => streamingMessageId.value),
    backendAvailable: computed(() => backendAvailable.value),

    createConversation,
    selectConversation,
    deleteConversation,
    updateConversationTitle,
    addMessage,
    updateMessage,
    sendMessage,
    clearAll,
    exportConversation,
    importConversation,
    loadFromStorage,
    loadConversations
  }
})