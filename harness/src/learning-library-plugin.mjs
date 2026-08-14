import { defineTool } from '@deepseek-ai/dsh-tools'
import { pluginName, registerLearningLibraryTools } from '../../packages/dsh-learning-library/src/core.mjs'

export const name = pluginName
export const inject = ['tools']

// Local adapter until the npm package is published. It keeps the project on
// the same generic bridge contract while resolving the Harness peer dependency
// from this deployment's node_modules.
export function apply(ctx, config = {}) {
  return registerLearningLibraryTools({ ctx, defineTool, config })
}
