import { describe, expect, it } from 'vitest'
import { CONCRETE_PLATFORM_OPTIONS, GROUP_PLATFORM_OPTIONS, ACCOUNT_PLATFORM_OPTIONS } from '@/constants/platforms'

const concretePlatforms = [
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'kimi',
  'zhipu',
  'deepseek',
  'minimax',
  'opencode_go'
]

describe('platform option catalogs', () => {
  it('lists asset providers as accounts without exposing them to LLM groups', () => {
    expect(ACCOUNT_PLATFORM_OPTIONS.map(option => option.value)).toEqual([...concretePlatforms, 'tripo', 'meshy'])
    expect(GROUP_PLATFORM_OPTIONS.some(option => ['tripo', 'meshy'].includes(option.value))).toBe(false)
  })
  it('exposes every concrete account platform', () => {
    expect(CONCRETE_PLATFORM_OPTIONS.map((option) => option.value)).toEqual(concretePlatforms)
  })

  it('adds composite for group-backed filters', () => {
    expect(GROUP_PLATFORM_OPTIONS.map((option) => option.value)).toEqual([
      ...concretePlatforms,
      'composite'
    ])
  })
})
