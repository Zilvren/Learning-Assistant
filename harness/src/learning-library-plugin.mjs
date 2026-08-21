import { defineTool } from '@deepseek-ai/dsh-tools'
import { pluginName, registerLearningLibraryTools } from '../../packages/dsh-learning-library/src/core.mjs'

export const name = pluginName
export const inject = ['tools']

// 在 npm 包发布前使用本地适配器。它让项目遵循相同的通用桥接约定，同时从本次部署的 node_modules 解析 Harness 对等依赖。
export function apply(ctx, config = {}) {
  return registerLearningLibraryTools({ ctx, defineTool, config })
}
