import { defineTool } from '@deepseek-ai/dsh-tools'
import { pluginName, registerLearningLibraryTools } from './core.mjs'

export const name = pluginName
export const inject = ['tools']

export function apply(ctx, config = {}) {
  return registerLearningLibraryTools({ ctx, defineTool, config })
}

export { createLearningLibraryBridge } from './client.mjs'
export { pluginName, registerLearningLibraryTools } from './core.mjs'
