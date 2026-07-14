<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Academic Research AI Platform -->
  <div v-else class="min-h-screen bg-white dark:bg-gray-950">

    <!-- ===== Navigation ===== -->
    <header class="sticky top-0 z-50 border-b border-gray-100/80 bg-white/90 backdrop-blur-md dark:border-gray-800/50 dark:bg-gray-950/90">
      <nav class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <!-- Logo -->
        <div class="flex items-center gap-3">
          <div class="h-9 w-9 overflow-hidden rounded-xl shadow-sm">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="text-base font-bold text-gray-900 dark:text-white">{{ siteName }}</span>
        </div>

        <!-- Center Nav Links -->
        <div class="hidden items-center gap-8 md:flex">
          <a href="#features" class="text-sm text-gray-600 transition-colors hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400">功能特性</a>
          <a href="#tools" class="text-sm text-gray-600 transition-colors hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400">研究工具</a>
          <a href="#api" class="text-sm text-gray-600 transition-colors hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400">API 集成</a>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-600 transition-colors hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400"
          >文档</a>
        </div>

        <!-- Right Actions -->
        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="ml-1 inline-flex items-center gap-2 rounded-full bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          >
            <span class="flex h-5 w-5 items-center justify-center rounded-full bg-white/20 text-xs font-bold">{{ userInitial }}</span>
            研究工作台
          </router-link>
          <router-link
            v-else
            to="/login"
            class="ml-1 inline-flex items-center rounded-full bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          >
            加入平台
          </router-link>
        </div>
      </nav>
    </header>

    <!-- ===== Hero Section ===== -->
    <section class="relative overflow-hidden bg-gradient-to-b from-blue-50/80 via-indigo-50/30 to-white px-6 py-28 dark:from-gray-900 dark:via-gray-900/60 dark:to-gray-950">
      <!-- Decorative blobs -->
      <div class="pointer-events-none absolute inset-0 overflow-hidden">
        <div class="absolute -top-32 left-1/3 h-[500px] w-[500px] rounded-full bg-blue-400/10 blur-3xl dark:bg-blue-500/5"></div>
        <div class="absolute bottom-0 right-1/4 h-[400px] w-[400px] rounded-full bg-indigo-400/10 blur-3xl dark:bg-indigo-500/5"></div>
        <div class="absolute top-1/4 right-1/3 h-[300px] w-[300px] rounded-full bg-violet-400/8 blur-3xl dark:bg-violet-500/5"></div>
      </div>

      <div class="relative mx-auto max-w-5xl text-center">
        <!-- Badge -->
        <div class="mb-8 inline-flex items-center gap-2 rounded-full border border-blue-200 bg-blue-50 px-4 py-1.5 text-sm font-medium text-blue-700 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-400">
          <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-blue-500"></span>
          AI 驱动的科研智能体平台
        </div>

        <!-- Main Heading -->
        <h1 class="mb-6 text-5xl font-extrabold tracking-tight text-gray-900 dark:text-white md:text-6xl lg:text-7xl">
          <span class="block">一站式</span>
          <span class="block bg-gradient-to-r from-blue-600 via-indigo-500 to-violet-600 bg-clip-text text-transparent">
            AI 科研工具平台
          </span>
        </h1>

        <!-- Subtitle -->
        <p class="mx-auto mb-4 max-w-2xl text-xl text-gray-600 dark:text-gray-400">
          {{ siteSubtitle }}
        </p>
        <p class="mx-auto mb-10 max-w-2xl text-base text-gray-500 dark:text-gray-500">
          覆盖文献阅读、学术搜索、数据分析、科研绘图与平台接入，
          让智能体贯穿科研全流程，大幅提升研究效率。
        </p>

        <!-- CTA Buttons -->
        <div class="flex flex-col items-center justify-center gap-4 sm:flex-row">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex items-center gap-2 rounded-full bg-blue-600 px-8 py-3.5 text-base font-semibold text-white shadow-lg shadow-blue-500/30 transition-all hover:-translate-y-0.5 hover:bg-blue-700 hover:shadow-xl hover:shadow-blue-500/30"
          >
            {{ isAuthenticated ? '进入控制台' : '免费开始使用' }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-2 rounded-full border border-gray-200 bg-white px-8 py-3.5 text-base font-medium text-gray-700 transition-all hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700"
          >
            查看文档
            <Icon name="book" size="sm" />
          </a>
        </div>

        <!-- Stats Row -->
        <div class="mx-auto mt-20 grid max-w-2xl grid-cols-3 gap-8">
          <div class="text-center">
            <div class="text-4xl font-extrabold text-gray-900 dark:text-white">5+</div>
            <div class="mt-1.5 text-sm text-gray-500 dark:text-gray-400">主流 AI 模型</div>
          </div>
          <div class="border-x border-gray-200 text-center dark:border-gray-800">
            <div class="text-4xl font-extrabold text-gray-900 dark:text-white">10+</div>
            <div class="mt-1.5 text-sm text-gray-500 dark:text-gray-400">科研工具</div>
          </div>
          <div class="text-center">
            <div class="text-4xl font-extrabold text-gray-900 dark:text-white">API</div>
            <div class="mt-1.5 text-sm text-gray-500 dark:text-gray-400">开发者友好</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== Features Section ===== -->
    <section id="features" class="bg-white px-6 py-24 dark:bg-gray-950">
      <div class="mx-auto max-w-7xl">
        <div class="mb-16 text-center">
          <h2 class="text-3xl font-bold text-gray-900 dark:text-white md:text-4xl">
            核心功能特性
          </h2>
          <p class="mt-4 text-lg text-gray-500 dark:text-gray-400">
            从文献阅读到数据分析，AI 贯穿科研全流程
          </p>
        </div>

        <div class="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          <!-- Feature: Literature -->
          <div class="group rounded-2xl border border-gray-100 bg-white p-8 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-blue-500/10 dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-blue-600 shadow-lg shadow-blue-500/30 transition-transform group-hover:scale-110">
              <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
              </svg>
            </div>
            <h3 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">智能文献阅读</h3>
            <p class="leading-relaxed text-gray-500 dark:text-gray-400">
              AI 辅助 PDF 解析、自动摘要提取、关键词高亮，快速理解文献核心内容，告别低效阅读。
            </p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">PDF 解析</span>
              <span class="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">自动摘要</span>
              <span class="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">关键词提取</span>
            </div>
          </div>

          <!-- Feature: Search -->
          <div class="group rounded-2xl border border-gray-100 bg-white p-8 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-indigo-500/10 dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-500 to-indigo-600 shadow-lg shadow-indigo-500/30 transition-transform group-hover:scale-110">
              <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
              </svg>
            </div>
            <h3 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">多源学术搜索</h3>
            <p class="leading-relaxed text-gray-500 dark:text-gray-400">
              整合多个学术数据库，智能检索相关文献，自动生成文献综述框架，加速研究选题。
            </p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">多库检索</span>
              <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">综述生成</span>
              <span class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">引用管理</span>
            </div>
          </div>

          <!-- Feature: Data Analysis -->
          <div class="group rounded-2xl border border-gray-100 bg-white p-8 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-violet-500/10 dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-violet-500 to-violet-600 shadow-lg shadow-violet-500/30 transition-transform group-hover:scale-110">
              <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
              </svg>
            </div>
            <h3 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">数据分析与可视化</h3>
            <p class="leading-relaxed text-gray-500 dark:text-gray-400">
              AI 辅助统计分析、自动生成科研图表，支持多种数据格式，让数据洞见一目了然。
            </p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-violet-50 px-3 py-1 text-xs font-medium text-violet-600 dark:bg-violet-900/30 dark:text-violet-400">统计分析</span>
              <span class="rounded-full bg-violet-50 px-3 py-1 text-xs font-medium text-violet-600 dark:bg-violet-900/30 dark:text-violet-400">图表生成</span>
              <span class="rounded-full bg-violet-50 px-3 py-1 text-xs font-medium text-violet-600 dark:bg-violet-900/30 dark:text-violet-400">数据导出</span>
            </div>
          </div>

          <!-- Feature: Writing -->
          <div class="group rounded-2xl border border-gray-100 bg-white p-8 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-teal-500/10 dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-teal-500 to-emerald-600 shadow-lg shadow-teal-500/30 transition-transform group-hover:scale-110">
              <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
              </svg>
            </div>
            <h3 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">学术写作辅助</h3>
            <p class="leading-relaxed text-gray-500 dark:text-gray-400">
              论文润色、格式规范检查、参考文献整理，全方位提升学术写作质量与发表成功率。
            </p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-600 dark:bg-teal-900/30 dark:text-teal-400">论文润色</span>
              <span class="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-600 dark:bg-teal-900/30 dark:text-teal-400">格式规范</span>
              <span class="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-600 dark:bg-teal-900/30 dark:text-teal-400">引用整理</span>
            </div>
          </div>

          <!-- Feature: API -->
          <div class="group rounded-2xl border border-gray-100 bg-white p-8 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-cyan-500/10 dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-cyan-500 to-blue-600 shadow-lg shadow-cyan-500/30 transition-transform group-hover:scale-110">
              <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5" />
              </svg>
            </div>
            <h3 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">API 开放集成</h3>
            <p class="leading-relaxed text-gray-500 dark:text-gray-400">
              OpenAI 兼容接口，一键接入多个主流 AI 模型，支持自定义开发与科研系统集成。
            </p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-cyan-50 px-3 py-1 text-xs font-medium text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-400">API Key</span>
              <span class="rounded-full bg-cyan-50 px-3 py-1 text-xs font-medium text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-400">多模型路由</span>
              <span class="rounded-full bg-cyan-50 px-3 py-1 text-xs font-medium text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-400">用量统计</span>
            </div>
          </div>

          <!-- Feature: Security -->
          <div class="group rounded-2xl border border-gray-100 bg-white p-8 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-amber-500/10 dark:border-gray-800 dark:bg-gray-900">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-amber-500 to-orange-600 shadow-lg shadow-amber-500/30 transition-transform group-hover:scale-110">
              <svg class="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6.11 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
              </svg>
            </div>
            <h3 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">安全可靠</h3>
            <p class="leading-relaxed text-gray-500 dark:text-gray-400">
              数据加密传输、细粒度权限控制，科研数据安全有保障，保护研究成果与知识产权。
            </p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span class="rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-600 dark:bg-amber-900/30 dark:text-amber-400">数据加密</span>
              <span class="rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-600 dark:bg-amber-900/30 dark:text-amber-400">权限隔离</span>
              <span class="rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-600 dark:bg-amber-900/30 dark:text-amber-400">用量监控</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== AI Models Section ===== -->
    <section id="tools" class="bg-gray-50 px-6 py-24 dark:bg-gray-900">
      <div class="mx-auto max-w-7xl">
        <div class="mb-16 text-center">
          <h2 class="text-3xl font-bold text-gray-900 dark:text-white md:text-4xl">
            接入主流 AI 模型
          </h2>
          <p class="mt-4 text-lg text-gray-500 dark:text-gray-400">
            一个 API Key，覆盖文本、代码、视觉、多模态全场景
          </p>
        </div>

        <div class="grid grid-cols-2 gap-4 md:grid-cols-4">
          <div
            v-for="capability in serviceCapabilities"
            :key="capability.key"
            class="group flex flex-col items-center gap-4 rounded-2xl border border-gray-200 bg-white p-6 text-center transition-all hover:-translate-y-0.5 hover:shadow-lg dark:border-gray-700 dark:bg-gray-800"
          >
            <div
              :class="[
                'flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br text-white shadow-lg transition-transform group-hover:scale-105',
                capability.gradient
              ]"
            >
              <Icon :name="capability.icon" size="md" :stroke-width="2" />
            </div>
            <div class="font-semibold text-gray-900 dark:text-white">{{ t(capability.labelKey) }}</div>
            <div class="inline-flex items-center gap-1 rounded-full bg-green-50 px-2.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">
              <span class="h-1.5 w-1.5 rounded-full bg-green-500"></span>
              已接入
            </div>
          </div>
        </div>

        <!-- Workflow steps -->
        <div class="mt-20 grid gap-6 md:grid-cols-4">
          <div v-for="(step, i) in workflowSteps" :key="i" class="relative text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-white shadow-md ring-4 ring-gray-50 dark:bg-gray-800 dark:ring-gray-900">
              <span class="text-2xl">{{ step.icon }}</span>
            </div>
            <h4 class="mb-2 font-semibold text-gray-900 dark:text-white">{{ step.title }}</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ step.desc }}</p>
            <!-- Connector arrow (hidden on last) -->
            <div v-if="i < workflowSteps.length - 1" class="absolute right-0 top-6 hidden -translate-y-1/2 translate-x-1/2 text-gray-300 md:block dark:text-gray-600">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" /></svg>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ===== CTA Section ===== -->
    <section id="api" class="relative overflow-hidden bg-gradient-to-r from-blue-600 to-indigo-600 px-6 py-24">
      <div class="pointer-events-none absolute inset-0 overflow-hidden">
        <div class="absolute -left-20 top-0 h-80 w-80 rounded-full bg-white/5 blur-3xl"></div>
        <div class="absolute -right-20 bottom-0 h-80 w-80 rounded-full bg-white/5 blur-3xl"></div>
      </div>
      <div class="relative mx-auto max-w-4xl text-center">
        <h2 class="mb-4 text-3xl font-bold text-white md:text-4xl">
          立即开启 AI 科研之旅
        </h2>
        <p class="mb-10 text-lg text-blue-100">
          注册即可体验智能文献阅读、数据分析等工具，
          开发者可直接通过 API Key 集成到现有工作流
        </p>
        <div class="flex flex-col items-center justify-center gap-4 sm:flex-row">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="inline-flex items-center gap-2 rounded-full bg-white px-8 py-3.5 text-base font-semibold text-blue-600 shadow-lg transition-all hover:-translate-y-0.5 hover:bg-blue-50 hover:shadow-xl"
          >
            {{ isAuthenticated ? '进入控制台' : '免费注册' }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <router-link
            to="/login"
            class="inline-flex items-center rounded-full border border-white/30 bg-white/10 px-8 py-3.5 text-base font-medium text-white transition-all hover:bg-white/20"
          >
            已有账号，直接登录
          </router-link>
        </div>
      </div>
    </section>

    <!-- ===== Footer ===== -->
    <footer class="border-t border-gray-100 bg-white px-6 py-12 dark:border-gray-800 dark:bg-gray-950">
      <div class="mx-auto max-w-7xl">
        <div class="flex flex-col items-center justify-between gap-6 md:flex-row">
          <div class="flex items-center gap-3">
            <div class="h-8 w-8 overflow-hidden rounded-lg">
              <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
            </div>
            <span class="font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
          </div>
          <div class="flex items-center gap-6">
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer"
              class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-white">
              文档
            </a>
            <a :href="githubUrl" target="_blank" rel="noopener noreferrer"
              class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-white">
              GitHub
            </a>
            <router-link to="/login"
              class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-white">
              登录
            </router-link>
          </div>
          <p class="text-sm text-gray-400 dark:text-gray-500">
            © {{ currentYear }} {{ siteName }}. All rights reserved.
          </p>
        </div>
      </div>
    </footer>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import { resolveDisplayBrandName } from '@/utils/branding'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

type ServiceCapability = {
  key: string
  labelKey: string
  icon: 'brain' | 'terminal' | 'eye' | 'sync'
  gradient: string
}

const { t, locale } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() =>
  resolveDisplayBrandName(appStore.cachedPublicSettings?.site_name || appStore.siteName, locale.value)
)
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() =>
  appStore.cachedPublicSettings?.site_subtitle || t('home.heroSubtitle')
)
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const serviceCapabilities: ServiceCapability[] = [
  {
    key: 'text',
    labelKey: 'home.providers.text',
    icon: 'brain',
    gradient: 'from-cyan-500 to-blue-600 shadow-cyan-500/30'
  },
  {
    key: 'coding',
    labelKey: 'home.providers.coding',
    icon: 'terminal',
    gradient: 'from-teal-500 to-emerald-600 shadow-teal-500/30'
  },
  {
    key: 'vision',
    labelKey: 'home.providers.vision',
    icon: 'eye',
    gradient: 'from-violet-500 to-fuchsia-600 shadow-violet-500/30'
  },
  {
    key: 'multimodal',
    labelKey: 'home.providers.multimodal',
    icon: 'sync',
    gradient: 'from-blue-500 to-indigo-600 shadow-blue-500/30'
  }
]

const workflowSteps = [
  { icon: '📚', title: '文献收集', desc: '多源检索，一键导入相关文献' },
  { icon: '🤖', title: 'AI 分析', desc: '自动解析、提取核心观点' },
  { icon: '📊', title: '数据可视化', desc: '生成图表，直观展现研究结果' },
  { icon: '✍️', title: '论文写作', desc: 'AI 辅助润色，提升写作效率' }
]

const isDark = ref(document.documentElement.classList.contains('dark'))
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})
const currentYear = computed(() => new Date().getFullYear())

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
/* smooth scrolling for anchor links */
html {
  scroll-behavior: smooth;
}
</style>