<template>
  <AppLayout>
    <div class="min-h-full bg-slate-50 px-4 py-6 dark:bg-gray-950 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-7xl space-y-6">
        <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900">
          <div class="relative p-6 sm:p-8">
            <div class="absolute inset-y-0 right-0 hidden w-1/2 bg-gradient-to-l from-blue-50 via-cyan-50/60 to-transparent dark:from-blue-950/30 dark:via-cyan-950/10 lg:block"></div>
            <div class="relative flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <p class="mb-2 text-sm font-semibold uppercase tracking-[0.24em] text-blue-600 dark:text-blue-400">Research Workspace</p>
                <h1 class="text-3xl font-bold text-slate-950 dark:text-white">工作空间与知识库</h1>
                <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 dark:text-gray-300">
                  用工作空间组织研究主题，把文档、笔记和提示素材沉淀为可复用知识。后续对话可以按工作空间绑定知识文件。
                </p>
              </div>
              <div class="flex flex-wrap gap-3">
                <button class="btn btn-secondary" :disabled="loading" @click="loadAll">
                  <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
                  刷新
                </button>
                <button class="btn btn-primary" @click="createWorkspaceQuick">
                  <Icon name="plus" size="md" />
                  新建工作空间
                </button>
              </div>
            </div>
          </div>
        </section>

        <div class="grid gap-6 lg:grid-cols-[320px_1fr]">
          <aside class="rounded-3xl border border-slate-200 bg-white p-4 shadow-sm dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="font-semibold text-slate-900 dark:text-white">工作空间</h2>
              <span class="rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-500 dark:bg-gray-800 dark:text-gray-400">{{ workspaces.length }}</span>
            </div>

            <div class="space-y-2">
              <button
                class="w-full rounded-2xl border px-4 py-3 text-left transition"
                :class="selectedWorkspaceId === null ? 'border-blue-300 bg-blue-50 text-blue-900 dark:border-blue-700 dark:bg-blue-950/30 dark:text-blue-100' : 'border-slate-200 hover:bg-slate-50 dark:border-gray-800 dark:hover:bg-gray-800'"
                @click="selectWorkspace(null)"
              >
                <div class="font-medium">全部知识</div>
                <div class="mt-1 text-xs text-slate-500 dark:text-gray-400">跨工作空间查看</div>
              </button>

              <button
                v-for="workspace in workspaces"
                :key="workspace.id"
                class="group w-full rounded-2xl border px-4 py-3 text-left transition"
                :class="selectedWorkspaceId === workspace.id ? 'border-blue-300 bg-blue-50 text-blue-900 dark:border-blue-700 dark:bg-blue-950/30 dark:text-blue-100' : 'border-slate-200 hover:bg-slate-50 dark:border-gray-800 dark:hover:bg-gray-800'"
                @click="selectWorkspace(workspace.id)"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <div class="truncate font-medium">{{ workspace.name }}</div>
                    <div class="mt-1 line-clamp-2 text-xs text-slate-500 dark:text-gray-400">{{ workspace.description || '暂无描述' }}</div>
                  </div>
                  <button class="rounded-lg p-1 text-slate-400 opacity-0 hover:bg-red-50 hover:text-red-600 group-hover:opacity-100 dark:hover:bg-red-950/30" @click.stop="deleteWorkspaceItem(workspace.id)">
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </button>
            </div>
          </aside>

          <main class="space-y-6">
            <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-900">
              <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 class="font-semibold text-slate-900 dark:text-white">知识文件</h2>
                  <p class="text-sm text-slate-500 dark:text-gray-400">当前 {{ filteredFiles.length }} 个文件</p>
                </div>
                <button class="btn btn-primary" :disabled="!workspaces.length" @click="openCreateFile">
                  <Icon name="plus" size="md" />
                  添加知识
                </button>
              </div>

              <div v-if="!workspaces.length" class="rounded-2xl border border-dashed border-slate-300 p-10 text-center dark:border-gray-700">
                <Icon name="database" size="xl" class="mx-auto mb-3 text-slate-400" />
                <h3 class="font-medium text-slate-900 dark:text-white">先创建一个工作空间</h3>
                <p class="mt-2 text-sm text-slate-500 dark:text-gray-400">工作空间是知识和对话的组织边界。</p>
              </div>

              <div v-else-if="filteredFiles.length === 0" class="rounded-2xl border border-dashed border-slate-300 p-10 text-center dark:border-gray-700">
                <Icon name="document" size="xl" class="mx-auto mb-3 text-slate-400" />
                <h3 class="font-medium text-slate-900 dark:text-white">暂无知识文件</h3>
                <p class="mt-2 text-sm text-slate-500 dark:text-gray-400">添加文本、笔记或研究材料，后续可用于对话上下文。</p>
              </div>

              <div v-else class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                <article v-for="file in filteredFiles" :key="file.id" class="rounded-2xl border border-slate-200 p-4 transition hover:-translate-y-0.5 hover:shadow-md dark:border-gray-800">
                  <div class="mb-3 flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <h3 class="truncate font-medium text-slate-900 dark:text-white">{{ file.filename }}</h3>
                      <p class="mt-1 text-xs text-slate-500 dark:text-gray-400">{{ file.file_type }} · {{ formatSize(file.file_size) }}</p>
                    </div>
                    <span class="rounded-full bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300">{{ file.status }}</span>
                  </div>
                  <p class="line-clamp-4 min-h-[5rem] whitespace-pre-wrap text-sm leading-5 text-slate-600 dark:text-gray-300">{{ file.content || '列表不加载正文，点击编辑查看。' }}</p>
                  <div class="mt-4 flex justify-end gap-2">
                    <button class="btn btn-secondary btn-sm" @click="editFile(file)">编辑</button>
                    <button class="btn btn-danger btn-sm" @click="deleteFileItem(file.id)">删除</button>
                  </div>
                </article>
              </div>
            </section>
          </main>
        </div>
      </div>
    </div>

    <div v-if="fileDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div class="w-full max-w-2xl rounded-3xl bg-white p-6 shadow-2xl dark:bg-gray-900">
        <div class="mb-5 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-slate-900 dark:text-white">{{ editingFile ? '编辑知识文件' : '添加知识文件' }}</h3>
          <button class="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-gray-800" @click="closeFileDialog"><Icon name="x" size="md" /></button>
        </div>
        <div class="space-y-4">
          <input v-model="fileForm.filename" class="input" placeholder="文件名，例如：量子纠缠实验笔记" />
          <select v-model="fileForm.workspace_id" class="input">
            <option v-for="workspace in workspaces" :key="workspace.id" :value="workspace.id">{{ workspace.name }}</option>
          </select>
          <input v-model="fileForm.file_type" class="input" placeholder="类型，例如：text / markdown / pdf" />
          <textarea v-model="fileForm.content" class="input min-h-[240px] resize-y" placeholder="粘贴或输入知识内容"></textarea>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="btn btn-secondary" @click="closeFileDialog">取消</button>
          <button class="btn btn-primary" :disabled="savingFile" @click="saveFile">保存</button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import workspaceKnowledgeAPI, { type KnowledgeFileDTO, type WorkspaceDTO } from '@/api/workspaceKnowledge'

const appStore = useAppStore()
const workspaces = ref<WorkspaceDTO[]>([])
const files = ref<KnowledgeFileDTO[]>([])
const selectedWorkspaceId = ref<number | null>(null)
const loading = ref(false)
const savingFile = ref(false)
const fileDialogOpen = ref(false)
const editingFile = ref<KnowledgeFileDTO | null>(null)

const fileForm = reactive({
  workspace_id: undefined as number | undefined,
  filename: '',
  file_type: 'text',
  content: ''
})

const filteredFiles = computed(() => files.value)

async function loadAll() {
  loading.value = true
  try {
    const workspaceResult = await workspaceKnowledgeAPI.listWorkspaces()
    workspaces.value = workspaceResult.items
    if (selectedWorkspaceId.value && !workspaces.value.some(w => w.id === selectedWorkspaceId.value)) {
      selectedWorkspaceId.value = null
    }
    await loadFiles()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载工作空间失败'))
  } finally {
    loading.value = false
  }
}

async function loadFiles() {
  const result = await workspaceKnowledgeAPI.listKnowledgeFiles(selectedWorkspaceId.value || undefined)
  files.value = await Promise.all(result.items.map(async item => {
    try {
      return await workspaceKnowledgeAPI.getKnowledgeFile(item.id)
    } catch {
      return item
    }
  }))
}

async function selectWorkspace(id: number | null) {
  selectedWorkspaceId.value = id
  try {
    await loadFiles()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载知识文件失败'))
  }
}

async function createWorkspaceQuick() {
  const name = window.prompt('工作空间名称', `科研工作空间 ${workspaces.value.length + 1}`)
  if (!name) return
  try {
    const workspace = await workspaceKnowledgeAPI.createWorkspace({ name, description: '用于组织对话、文档和知识素材' })
    workspaces.value.unshift(workspace)
    selectedWorkspaceId.value = workspace.id
    appStore.showSuccess('工作空间已创建')
    await loadFiles()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '创建工作空间失败'))
  }
}

async function deleteWorkspaceItem(id: number) {
  if (!confirm('确定删除这个工作空间吗？知识文件会保留但解除归属。')) return
  try {
    await workspaceKnowledgeAPI.deleteWorkspace(id)
    workspaces.value = workspaces.value.filter(item => item.id !== id)
    if (selectedWorkspaceId.value === id) selectedWorkspaceId.value = null
    appStore.showSuccess('工作空间已删除')
    await loadFiles()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除工作空间失败'))
  }
}

function openCreateFile() {
  editingFile.value = null
  fileForm.workspace_id = selectedWorkspaceId.value || workspaces.value[0]?.id
  fileForm.filename = ''
  fileForm.file_type = 'text'
  fileForm.content = ''
  fileDialogOpen.value = true
}

function editFile(file: KnowledgeFileDTO) {
  editingFile.value = file
  fileForm.workspace_id = file.workspace_id || selectedWorkspaceId.value || workspaces.value[0]?.id
  fileForm.filename = file.filename
  fileForm.file_type = file.file_type
  fileForm.content = file.content || ''
  fileDialogOpen.value = true
}

function closeFileDialog() {
  fileDialogOpen.value = false
  editingFile.value = null
}

async function saveFile() {
  if (!fileForm.filename.trim() || !fileForm.content.trim()) {
    appStore.showError('文件名和内容不能为空')
    return
  }
  savingFile.value = true
  try {
    if (editingFile.value) {
      await workspaceKnowledgeAPI.updateKnowledgeFile(editingFile.value.id, { ...fileForm })
      appStore.showSuccess('知识文件已更新')
    } else {
      await workspaceKnowledgeAPI.createKnowledgeFile({ ...fileForm, workspace_id: fileForm.workspace_id })
      appStore.showSuccess('知识文件已创建')
    }
    closeFileDialog()
    await loadFiles()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存知识文件失败'))
  } finally {
    savingFile.value = false
  }
}

async function deleteFileItem(id: number) {
  if (!confirm('确定删除这个知识文件吗？')) return
  try {
    await workspaceKnowledgeAPI.deleteKnowledgeFile(id)
    files.value = files.value.filter(item => item.id !== id)
    appStore.showSuccess('知识文件已删除')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除知识文件失败'))
  }
}

function formatSize(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

onMounted(loadAll)
</script>