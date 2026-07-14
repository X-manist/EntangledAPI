<template>
  <div class="flex h-screen overflow-hidden bg-white dark:bg-gray-950">
    <!-- Sidebar -->
    <div
      :class="[
        'flex flex-col border-r border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900',
        isSidebarOpen ? 'w-64' : 'w-0 md:w-64'
      ]"
      class="transition-all duration-300"
    >
      <!-- Sidebar Header -->
      <div class="flex items-center justify-between border-b border-gray-200 p-4 dark:border-gray-800">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">对话历史</h2>
        <button
          @click="handleNewChat"
          class="rounded-lg bg-blue-600 p-2 text-white hover:bg-blue-700 transition-colors"
          title="新对话"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>

      <!-- Conversations List -->
      <div class="flex-1 overflow-y-auto p-2">
        <div v-if="sortedConversations.length === 0" class="p-4 text-center text-sm text-gray-500 dark:text-gray-400">
          暂无对话记录
        </div>
        <button
          v-for="conv in sortedConversations"
          :key="conv.id"
          @click="chatStore.selectConversation(conv.id)"
          :class="[
            'group relative mb-2 w-full rounded-lg p-3 text-left transition-colors',
            activeConversationId === conv.id
              ? 'bg-white shadow-sm dark:bg-gray-800'
              : 'hover:bg-white/50 dark:hover:bg-gray-800/50'
          ]"
        >
          <div class="mb-1 truncate font-medium text-gray-900 dark:text-white">
            {{ conv.title }}
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ formatDate(conv.updatedAt) }}
          </div>
          
          <!-- Delete button -->
          <button
            @click.stop="handleDelete(conv.id)"
            class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-gray-400 opacity-0 transition-opacity hover:bg-red-50 hover:text-red-600 group-hover:opacity-100 dark:hover:bg-red-900/20"
            title="删除"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </button>
      </div>

      <!-- Sidebar Footer -->
      <div class="border-t border-gray-200 p-3 dark:border-gray-800">
        <button
          @click="handleClearAll"
          class="w-full rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-800 transition-colors"
        >
          清空所有对话
        </button>
      </div>
    </div>

    <!-- Main Chat Area -->
    <div class="flex flex-1 flex-col">
      <!-- Chat Header -->
      <div class="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-4 dark:border-gray-800 dark:bg-gray-950">
        <div class="flex items-center gap-3">
          <button
            @click="isSidebarOpen = !isSidebarOpen"
            class="rounded-lg p-2 hover:bg-gray-100 dark:hover:bg-gray-800 md:hidden"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ activeConversation?.title || 'AI 科研助手' }}
          </h1>
        </div>
        
        <div class="flex items-center gap-2">
          <router-link
            to="/dashboard"
            class="rounded-lg px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
          >
            返回控制台
          </router-link>
        </div>
      </div>

      <!-- Messages Area -->
      <div ref="messagesContainer" class="flex-1 overflow-y-auto px-6 py-6">
        <div v-if="!activeConversation" class="flex h-full items-center justify-center">
          <div class="text-center">
            <div class="mb-4 text-6xl">💬</div>
            <h2 class="mb-2 text-xl font-semibold text-gray-900 dark:text-white">开始新对话</h2>
            <p class="text-gray-500 dark:text-gray-400">点击左上角按钮创建对话</p>
          </div>
        </div>

        <div v-else class="mx-auto max-w-3xl space-y-6">
          <div
            v-for="message in activeConversation.messages"
            :key="message.id"
            :class="[
              'flex gap-4',
              message.role === 'user' ? 'justify-end' : 'justify-start'
            ]"
          >
            <!-- Avatar -->
            <div
              v-if="message.role === 'assistant'"
              class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-white"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
              </svg>
            </div>

            <!-- Message Bubble -->
            <div
              :class="[
                'max-w-[80%] rounded-2xl px-4 py-3',
                message.role === 'user'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-white'
              ]"
            >
              <div class="prose prose-sm max-w-none dark:prose-invert" v-html="renderMarkdown(message.content)"></div>
              
              <div v-if="message.tokens || message.cost" class="mt-2 flex items-center gap-3 text-xs opacity-70">
                <span v-if="message.tokens">{{ message.tokens }} tokens</span>
                <span v-if="message.cost">¥{{ message.cost.toFixed(4) }}</span>
              </div>
            </div>

            <!-- User Avatar -->
            <div
              v-if="message.role === 'user'"
              class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
            >
              {{ userInitial }}
            </div>
          </div>

          <!-- Streaming indicator -->
          <div v-if="chatStore.isStreaming" class="flex gap-4">
            <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-white">
              <svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            </div>
            <div class="flex items-center gap-1 text-gray-500">
              <span class="h-2 w-2 animate-bounce rounded-full bg-gray-400"></span>
              <span class="h-2 w-2 animate-bounce rounded-full bg-gray-400" style="animation-delay: 0.1s"></span>
              <span class="h-2 w-2 animate-bounce rounded-full bg-gray-400" style="animation-delay: 0.2s"></span>
            </div>
          </div>
        </div>
      </div>

      <!-- Input Area -->
      <div class="border-t border-gray-200 bg-white px-6 py-4 dark:border-gray-800 dark:bg-gray-950">
        <div class="mx-auto max-w-3xl">
          <div class="relative">
            <textarea
              v-model="inputMessage"
              @keydown.enter.exact.prevent="handleSend"
              @keydown.meta.enter="handleSend"
              @keydown.ctrl.enter="handleSend"
              :disabled="chatStore.isStreaming || !activeConversation"
              placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
              rows="3"
              class="w-full resize-none rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 pr-12 text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:placeholder-gray-500"
            ></textarea>
            
            <button
              @click="handleSend"
              :disabled="!inputMessage.trim() || chatStore.isStreaming || !activeConversation"
              class="absolute bottom-3 right-3 rounded-lg bg-blue-600 p-2 text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            </button>
          </div>
          
          <div class="mt-2 text-center text-xs text-gray-500 dark:text-gray-400">
            AI 可能会出错，请核查重要信息
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useAuthStore } from '@/stores/auth'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const router = useRouter()
const chatStore = useChatStore()
const authStore = useAuthStore()

const isSidebarOpen = ref(true)
const inputMessage = ref('')
const messagesContainer = ref<HTMLElement | null>(null)

const sortedConversations = computed(() => chatStore.sortedConversations)
const activeConversation = computed(() => chatStore.activeConversation)
const activeConversationId = computed(() => chatStore.activeConversationId)
const userInitial = computed(() => {
  const email = authStore.user?.email
  return email ? email.charAt(0).toUpperCase() : 'U'
})

function renderMarkdown(content: string): string {
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
}

function formatDate(timestamp: number): string {
  const now = Date.now()
  const diff = now - timestamp
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)} 天前`
  
  return new Date(timestamp).toLocaleDateString('zh-CN', {
    month: 'numeric',
    day: 'numeric'
  })
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

async function handleNewChat() {
  await chatStore.createConversation()
  inputMessage.value = ''
}

async function handleDelete(id: string) {
  if (confirm('确定要删除这个对话吗？')) {
    await chatStore.deleteConversation(id)
  }
}

async function handleClearAll() {
  if (confirm('确定要清空所有对话吗？此操作无法撤销。')) {
    await chatStore.clearAll()
  }
}

async function handleSend() {
  if (!inputMessage.value.trim() || chatStore.isStreaming || !activeConversation.value) {
    return
  }

  const message = inputMessage.value.trim()
  inputMessage.value = ''

  try {
    await chatStore.sendMessage(activeConversation.value.id, message)
    scrollToBottom()
  } catch (error: any) {
    console.error('Send failed:', error)
    alert(error.message || '发送失败，请重试')
  }
}

// Auto-scroll when new messages arrive
watch(
  () => activeConversation.value?.messages.length,
  () => scrollToBottom(),
  { flush: 'post' }
)

// Create initial conversation if none exists
onMounted(async () => {
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }

  await chatStore.loadConversations()
  if (sortedConversations.value.length === 0) {
    await handleNewChat()
  }
})
</script>

<style scoped>
/* Markdown prose styling is handled by Tailwind's typography plugin */
</style>