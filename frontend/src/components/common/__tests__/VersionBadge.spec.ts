import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { execFileSync } from 'node:child_process'
import { chmodSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const mocks = vi.hoisted(() => ({
  getRollbackVersions: vi.fn(),
  copyToClipboard: vi.fn(),
  appStore: {
    versionLoading: false,
    currentVersion: '0.1.153-entangled.2',
    latestVersion: '0.1.153-entangled.2',
    hasUpdate: false,
    buildType: 'release',
    releaseInfo: null,
    updateRepository: 'X-manist/EntangledAPI',
    updateDockerImage: 'ghcr.io/x-manist/sub2api',
    fetchVersion: vi.fn(),
    clearVersionCache: vi.fn()
  }
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAdmin: true }),
  useAppStore: () => mocks.appStore
}))

vi.mock('@/api/admin/system', () => ({
  performUpdate: vi.fn(),
  restartService: vi.fn(),
  rollback: vi.fn(),
  getRollbackVersions: mocks.getRollbackVersions
}))

vi.mock('@/composables/useClipboard', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useClipboard: () => ({
      copied: ref(false),
      copyToClipboard: mocks.copyToClipboard
    })
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import VersionBadge from '../VersionBadge.vue'

async function openRollbackCommand(wrapper: VueWrapper) {
  await wrapper.find('button').trigger('click')
  const rollbackButton = wrapper.findAll('button').find((button) =>
    button.text().includes('version.rollback')
  )
  expect(rollbackButton).toBeTruthy()
  await rollbackButton!.trigger('click')
  await flushPromises()

  const versionButton = wrapper.findAll('button').find((button) =>
    button.text().includes('v0.1.152-entangled.1')
  )
  expect(versionButton).toBeTruthy()
  await versionButton!.trigger('click')
  await flushPromises()
}

describe('VersionBadge release source commands', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(mocks.appStore, {
      updateRepository: 'X-manist/EntangledAPI',
      updateDockerImage: 'ghcr.io/x-manist/sub2api'
    })
    mocks.getRollbackVersions.mockResolvedValue({
      versions: [
        {
          version: '0.1.152-entangled.1',
          published_at: '2026-07-18T00:00:00Z',
          html_url: 'https://github.com/X-manist/EntangledAPI/releases/tag/v0.1.152-entangled.1'
        }
      ]
    })
  })

  it('keeps public and private custom-repository rollback commands independent', async () => {
    const wrapper = mount(VersionBadge, {
      global: {
        stubs: { Icon: true }
      }
    })
    await openRollbackCommand(wrapper)

    const publicCommand = wrapper.find('code').text()
    expect(publicCommand).toContain(
      'raw.githubusercontent.com/X-manist/EntangledAPI/v0.1.152-entangled.1/deploy/install.sh'
    )
    expect(publicCommand).not.toContain(
      'api.github.com/repos/X-manist/EntangledAPI/contents/deploy/install.sh'
    )
    expect(publicCommand).not.toContain('| sudo')
    expect(() => execFileSync('/bin/sh', ['-n'], { input: publicCommand })).not.toThrow()

    const copyButton = wrapper.findAll('button').find((button) =>
      button.text().includes('version.copyCommand')
    )
    expect(copyButton).toBeTruthy()
    await copyButton!.trigger('click')
    expect(mocks.copyToClipboard).toHaveBeenLastCalledWith(publicCommand)

    const privateTab = wrapper.findAll('button').find((button) =>
      button.text().includes('version.deployScriptPrivate')
    )
    expect(privateTab).toBeTruthy()
    await privateTab!.trigger('click')

    const privateCommand = wrapper.find('code').text()
    expect(privateCommand).toContain(
      'api.github.com/repos/X-manist/EntangledAPI/contents/deploy/install.sh'
    )
    expect(privateCommand).not.toContain('raw.githubusercontent.com/X-manist/EntangledAPI')
    expect(privateCommand).toContain('GITHUB_TOKEN:?Export a read-only GITHUB_TOKEN first')
    expect(privateCommand).toContain('GITHUB_TOKEN="$GITHUB_TOKEN"')
    expect(() => execFileSync('/bin/sh', ['-n'], { input: privateCommand })).not.toThrow()
    await copyButton!.trigger('click')
    expect(mocks.copyToClipboard).toHaveBeenLastCalledWith(privateCommand)
    wrapper.unmount()
  })

  it('builds an official rollback command with explicit source flags', async () => {
    Object.assign(mocks.appStore, {
      updateRepository: 'Wei-Shaw/sub2api',
      updateDockerImage: 'weishaw/sub2api'
    })
    const wrapper = mount(VersionBadge, {
      global: {
        stubs: { Icon: true }
      }
    })
    await openRollbackCommand(wrapper)

    const command = wrapper.find('code').text()
    expect(command).toContain('raw.githubusercontent.com/Wei-Shaw/sub2api')
    expect(command).toContain("--channel 'official'")
    expect(command).toContain("--repository 'Wei-Shaw/sub2api'")
    expect(
      wrapper.findAll('button').some((button) =>
        button.text().includes('version.deployScriptPrivate')
      )
    ).toBe(false)

    const dockerTab = wrapper.findAll('button').find((button) =>
      button.text().includes('version.deployDocker')
    )
    expect(dockerTab).toBeTruthy()
    await dockerTab!.trigger('click')
    const dockerCommand = wrapper.find('code').text()
    expect(dockerCommand).toContain(
      "set_env SUB2API_IMAGE 'weishaw/sub2api:0.1.152-entangled.1'"
    )
    expect(dockerCommand).toContain("set_env UPDATE_REPOSITORY 'Wei-Shaw/sub2api'")
    expect(dockerCommand).toContain("set_env UPDATE_GITHUB_TOKEN ''")
    wrapper.unmount()
  })

  it('keeps Docker image and update repository on the same channel', async () => {
    const wrapper = mount(VersionBadge, {
      global: {
        stubs: { Icon: true }
      }
    })
    await openRollbackCommand(wrapper)

    const dockerTab = wrapper.findAll('button').find((button) =>
      button.text().includes('version.deployDocker')
    )
    expect(dockerTab).toBeTruthy()
    await dockerTab!.trigger('click')

    const command = wrapper.find('code').text()
    expect(command).toContain(
      "set_env SUB2API_IMAGE 'ghcr.io/x-manist/sub2api:0.1.152-entangled.1'"
    )
    expect(command).toContain("set_env UPDATE_REPOSITORY 'X-manist/EntangledAPI'")
    expect(command).toContain("set_env UPDATE_DOCKER_IMAGE 'ghcr.io/x-manist/sub2api'")
    expect(command).toContain("set_env SUB2API_RUNTIME_SEED_POLICY 'if-missing'")
    expect(command).toContain('docker compose pull sub2api')
    expect(command).toContain(
      'SUB2API_RUNTIME_SEED_POLICY=always docker compose up -d --force-recreate --wait sub2api'
    )
    expect(command.match(/docker compose up -d --force-recreate --wait sub2api/g)).toHaveLength(2)

    const workDir = mkdtempSync(join(tmpdir(), 'sub2api-version-command-'))
    try {
      const fakeBin = join(workDir, 'bin')
      const dockerLog = join(workDir, 'docker.log')
      execFileSync('/bin/mkdir', ['-p', fakeBin])
      const fakeDocker = join(fakeBin, 'docker')
      writeFileSync(
        fakeDocker,
        '#!/bin/sh\nprintf \'%s|%s\\n\' "${SUB2API_RUNTIME_SEED_POLICY:-}" "$*" >> "$DOCKER_LOG"\n'
      )
      chmodSync(fakeDocker, 0o755)

      execFileSync('/bin/sh', ['-c', command], {
        cwd: workDir,
        env: {
          ...process.env,
          PATH: `${fakeBin}:${process.env.PATH || ''}`,
          DOCKER_LOG: dockerLog
        }
      })

      const persistedEnv = readFileSync(join(workDir, '.env'), 'utf8')
      expect(persistedEnv).toContain(
        'SUB2API_IMAGE=ghcr.io/x-manist/sub2api:0.1.152-entangled.1'
      )
      expect(persistedEnv).toContain('UPDATE_REPOSITORY=X-manist/EntangledAPI')
      expect(persistedEnv).toContain('UPDATE_DOCKER_IMAGE=ghcr.io/x-manist/sub2api')
      expect(persistedEnv).toContain('SUB2API_RUNTIME_SEED_POLICY=if-missing')

      const dockerCalls = readFileSync(dockerLog, 'utf8')
      expect(dockerCalls).toContain('|compose pull sub2api')
      expect(dockerCalls).toContain(
        'always|compose up -d --force-recreate --wait sub2api'
      )
    } finally {
      rmSync(workDir, { recursive: true, force: true })
    }
    wrapper.unmount()
  })
})
