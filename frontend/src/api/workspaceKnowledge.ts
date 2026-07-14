import { apiClient } from './client'

export interface WorkspaceDTO {
  id: number
  name: string
  description: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface KnowledgeFileDTO {
  id: number
  workspace_id?: number
  filename: string
  file_type: string
  file_size: number
  content?: string
  content_hash?: string
  metadata: Record<string, unknown>
  status: 'uploaded' | 'processing' | 'ready' | 'failed'
  created_at: string
  updated_at: string
}

export interface PaginatedResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface CreateWorkspacePayload {
  name: string
  description?: string
  metadata?: Record<string, unknown>
}

export interface UpdateWorkspacePayload {
  name?: string
  description?: string
  metadata?: Record<string, unknown>
}

export interface CreateKnowledgeFilePayload {
  workspace_id?: number
  filename: string
  file_type?: string
  content: string
  metadata?: Record<string, unknown>
}

export interface UpdateKnowledgeFilePayload {
  workspace_id?: number
  filename?: string
  file_type?: string
  content?: string
  status?: KnowledgeFileDTO['status']
  metadata?: Record<string, unknown>
}

export async function listWorkspaces(): Promise<PaginatedResult<WorkspaceDTO>> {
  const { data } = await apiClient.get<PaginatedResult<WorkspaceDTO>>('/workspaces', {
    params: { page: 1, page_size: 100 }
  })
  return data
}

export async function createWorkspace(payload: CreateWorkspacePayload): Promise<WorkspaceDTO> {
  const { data } = await apiClient.post<WorkspaceDTO>('/workspaces', payload)
  return data
}

export async function updateWorkspace(id: number, payload: UpdateWorkspacePayload): Promise<WorkspaceDTO> {
  const { data } = await apiClient.put<WorkspaceDTO>(`/workspaces/${id}`, payload)
  return data
}

export async function deleteWorkspace(id: number): Promise<void> {
  await apiClient.delete(`/workspaces/${id}`)
}

export async function listKnowledgeFiles(workspaceId?: number): Promise<PaginatedResult<KnowledgeFileDTO>> {
  const { data } = await apiClient.get<PaginatedResult<KnowledgeFileDTO>>('/knowledge-files', {
    params: { page: 1, page_size: 100, workspace_id: workspaceId }
  })
  return data
}

export async function getKnowledgeFile(id: number): Promise<KnowledgeFileDTO> {
  const { data } = await apiClient.get<KnowledgeFileDTO>(`/knowledge-files/${id}`)
  return data
}

export async function createKnowledgeFile(payload: CreateKnowledgeFilePayload): Promise<KnowledgeFileDTO> {
  const { data } = await apiClient.post<KnowledgeFileDTO>('/knowledge-files', payload)
  return data
}

export async function updateKnowledgeFile(id: number, payload: UpdateKnowledgeFilePayload): Promise<KnowledgeFileDTO> {
  const { data } = await apiClient.put<KnowledgeFileDTO>(`/knowledge-files/${id}`, payload)
  return data
}

export async function deleteKnowledgeFile(id: number): Promise<void> {
  await apiClient.delete(`/knowledge-files/${id}`)
}

export const workspaceKnowledgeAPI = {
  listWorkspaces,
  createWorkspace,
  updateWorkspace,
  deleteWorkspace,
  listKnowledgeFiles,
  getKnowledgeFile,
  createKnowledgeFile,
  updateKnowledgeFile,
  deleteKnowledgeFile
}

export default workspaceKnowledgeAPI