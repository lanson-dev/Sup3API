import { mount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { reactive } from 'vue'
import ConnectView from '../ConnectView.vue'
import ConsoleView from '../ConsoleView.vue'

const api = vi.hoisted(() => ({ list:vi.fn(), groups:vi.fn(), create:vi.fn(), toggle:vi.fn(), stats:vi.fn(), usage:vi.fn() }))
const route = reactive({path:'/keys'})
vi.mock('vue-router',()=>({useRoute:()=>route}))
vi.mock('@/stores/auth',()=>({useAuthStore:()=>({user:{balance:12},isAuthenticated:true})}))
vi.mock('@/api/keys',()=>({keysAPI:{list:api.list,create:api.create,toggleStatus:api.toggle}}))
vi.mock('@/api/groups',()=>({userGroupsAPI:{getAvailable:api.groups}}))
vi.mock('@/api/usage',()=>({usageAPI:{getDashboardStats:api.stats,list:api.usage}}))
const global = { stubs:{Sup3APIShell:{template:'<div><slot /></div>'},RouterLink:{template:'<a><slot /></a>'}} }
beforeEach(()=>{vi.clearAllMocks();route.path='/keys'})

describe('Sup3API customer portal',()=>{
  it('renders masked real account keys and requires explicit creation',async()=>{
    api.list.mockResolvedValue({items:[{id:7,name:'game-service',key:'ag-private-secret-for-test-only',status:'active',quota_used:0,quota:10,expires_at:null}],total:1})
    api.groups.mockResolvedValue([{id:3,name:'GPT',platform:'openai'}])
    api.create.mockResolvedValue({key:'ag-new-private-key-for-test'})
    const wrapper=mount(ConsoleView,{global});await flushPromises()
    expect(wrapper.text()).toContain('game-service')
    expect(wrapper.text()).not.toContain('ag-private-secret-for-test-only')
    expect(api.create).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(b=>b.text().includes('创建密钥'))!.trigger('click')
    await wrapper.find('input[placeholder="例如：game-asset-service"]').setValue('asset-pipeline')
    await wrapper.find('select').setValue('3')
    await wrapper.find('form').trigger('submit');await flushPromises()
    expect(api.create).toHaveBeenCalledWith('asset-pipeline',3,undefined,undefined,undefined,0,undefined)
    expect(wrapper.text()).not.toContain('ag-new-private-key-for-test')
    wrapper.unmount()
  })
  it('only sends explicit diagnostics to same-origin endpoints, never inserts keys into snippets',async()=>{
    const fetch=vi.fn().mockResolvedValue({ok:true,headers:new Headers({'content-type':'application/json'}),json:async()=>({providers:[]})})
    vi.stubGlobal('fetch',fetch)
    const wrapper=mount(ConnectView,{global})
    expect(fetch).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(b=>b.text()==='Meshy · 3D')!.trigger('click')
    await wrapper.find('input[type="password"]').setValue('private-key-not-for-examples')
    expect(wrapper.find('pre').text()).not.toContain('private-key-not-for-examples')
    await wrapper.findAll('button').find(b=>b.text()==='查询资产能力')!.trigger('click');await flushPromises()
    expect(fetch).toHaveBeenCalledWith('/v1/assets/capabilities',expect.objectContaining({method:'GET',credentials:'omit'}))
    await wrapper.findAll('button').find(b=>b.text()==='校验与估价')!.trigger('click');await flushPromises()
    expect(fetch).toHaveBeenLastCalledWith('/v1/assets/quotes',expect.objectContaining({method:'POST'}))
    expect(fetch.mock.calls.some(([url])=>url==='/v1/assets/jobs')).toBe(false)
    wrapper.unmount();vi.unstubAllGlobals()
  })
})
