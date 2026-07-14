import { apiClient } from './client'

export type ChatRole = 'user' | 'assistant' | 'system'

export interface ChatMessageDTO {
  id: number
  conversation_id: number
  role: ChatRole
  content: string
  attachments: Array<Record<string, unknown>>
  tokens_used: number
  cost: number
  model?: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ChatConversationDTO {
  id: number
  title: string
  knowledge_ids: number[]
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
  messages?: ChatMessageDTO[]
}

export interface PaginatedChatConversations {
  items: ChatConversationDTO[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface CreateConversationPayload {
  title?: string
  knowledge_ids?: number[]
  metadata?: Record<string, unknown>
}

export interface UpdateConversationPayload {
  title?: string
  knowledge_ids?: number[]
  metadata?: Record<string, unknown>
}

export interface CreateMessagePayload {
  role: ChatRole
  content: string
  attachments?: Array<Record<string, unknown>>
  tokens_used?: number
  cost?: number
  model?: string
  metadata?: Record<string, unknown>
}

export async function listConversations(): Promise<PaginatedChatConversations> {
  const { data } = await apiClient.get<PaginatedChatConversations>('/chat/conversations', {
    params: { page: 1, page_size: 100 }
  })
  return data
}

export async function getConversation(id: string | number): Promise<ChatConversationDTO> {
  const { data } = await apiClient.get<ChatConversationDTO>(`/chat/conversations/${id}`)
  return data
}

export async function createConversation(payload: CreateConversationPayload = {}): Promise<ChatConversationDTO> {
  const { data } = await apiClient.post<ChatConversationDTO>('/chat/conversations', payload)
  return data
}

export async function updateConversation(
  id: string | number,
  payload: UpdateConversationPayload
): Promise<ChatConversationDTO> {
  const { data } = await apiClient.put<ChatConversationDTO>(`/chat/conversations/${id}`, payload)
  return data
}

export async function deleteConversation(id: string | number): Promise<void> {
  await apiClient.delete(`/chat/conversations/${id}`)
}

export async function createMessage(
  conversationId: string | number,
  payload: CreateMessagePayload
): Promise<ChatMessageDTO> {
  const { data } = await apiClient.post<ChatMessageDTO>(`/chat/conversations/${conversationId}/messages`, payload)
  return data
}

export const chatAPI = {
  listConversations,
  getConversation,
  createConversation,
  updateConversation,
  deleteConversation,
  createMessage
}

export default chatAPI