import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AssetAccountModal from '../AssetAccountModal.vue'
import type { Account } from '@/types'

const api = vi.hoisted(() => ({ create: vi.fn().mockResolvedValue({}), update: vi.fn().mockResolvedValue({}) }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: api } }))
const global = { stubs: { AssetModelSettings: true, GroupSelector: true, BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' } } }

describe('upstream asset accounts', () => {
  it('creates a provider account through admin CRUD, not application keys', async () => {
    const wrapper = mount(AssetAccountModal, { props: { show: false, groups: [], provider: 'meshy', account: null }, global })
    await wrapper.setProps({ show: true })
    await wrapper.get('input[maxlength]').setValue('Meshy production')
    await wrapper.get('input[type=password]').setValue('upstream-test-key')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(api.create).toHaveBeenCalledWith(expect.objectContaining({ platform: 'meshy', type: 'apikey', credentials: { api_key: 'upstream-test-key', model_mapping: {} } }))
    expect((wrapper.get('input[type=password]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('never loads stored credentials into the form and preserves them on an ordinary edit', async () => {
    const account = { id: 9, name: 'Tripo', platform: 'tripo', status: 'active', priority: 1, credentials: { api_key: 'stored-secret' } } as Account
    const wrapper = mount(AssetAccountModal, { props: { show: false, groups: [], provider: 'tripo', account }, global })
    await wrapper.setProps({ show: true })
    expect((wrapper.get('input[type=password]').element as HTMLInputElement).value).toBe('')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(api.update).toHaveBeenCalledWith(9, { name: 'Tripo', priority: 1, status: 'active', group_ids: [], credentials: { model_mapping: {} } })
  })
})
