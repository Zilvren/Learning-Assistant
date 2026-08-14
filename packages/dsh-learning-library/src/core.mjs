import { createLearningLibraryBridge } from './client.mjs'

export const pluginName = 'dsh-learning-library'

function jsonOutput() {
  return {
    schema: { type: 'object', additionalProperties: true },
    render: (_args, value) => [{ type: 'text', text: JSON.stringify(value) }],
  }
}

/**
 * Registers five constrained learning-library tools. The package has no
 * filesystem, shell, or generic database capability. The host bridge owns
 * scope and may allow a new note to be created or an exact current version to
 * be updated; it never exposes force-overwrite, move, or delete operations.
 */
export function registerLearningLibraryTools({ ctx, defineTool, config = {}, dependencies = {} }) {
  if (!ctx?.tools?.register || typeof defineTool !== 'function') {
    throw new Error('dsh-learning-library requires the DeepSeek Harness tools service')
  }

  const bridge = createLearningLibraryBridge(config, dependencies)
  const register = (name, description, parameters, bridgeTool) => {
    ctx.tools.register(defineTool({
      name,
      description,
      parameters,
      output: jsonOutput(),
      execute: (args) => bridge.call(bridgeTool, args),
    }))
  }

  register(
    'list_library_paths',
    'List readable folders and notes within the current conversation\'s allowed learning-library scope. Use it to disambiguate a path before reading, creating, or updating a note.',
    {
      query: { type: 'string', description: 'Optional words used to narrow returned paths.' },
      limit: { type: 'number', description: 'Maximum paths to return, between 1 and 80.' },
    },
    'list_paths',
  )

  register(
    'search_library',
    'Search the allowed learning-library scope for relevant notes. Returns titles, paths, tags, and short excerpts; call read_library_note only for notes you need in full.',
    {
      query: { type: 'string', required: true, description: 'Learning topic, question, or keywords to search for.' },
      limit: { type: 'number', description: 'Maximum results to return, between 1 and 12.' },
    },
    'search',
  )

  register(
    'read_library_note',
    'Read one readable note from the current allowed learning-library scope by its numeric id. The result includes current_version, which is required to update an existing note safely. Never request files outside the scope.',
    {
      item_id: { type: 'number', required: true, description: 'The note id returned by list_library_paths or search_library.' },
    },
    'read_note',
  )

  register(
    'create_library_note',
    'Immediately create a new Markdown or text note at an explicit path. Before calling, resolve an ambiguous path with list_library_paths; when the user gives only a folder, choose a concise, meaningful .md filename from the requested content. This never overwrites an existing item; if the path already exists, read that note and use update_library_note instead. Only say the note was created after this tool succeeds.',
    {
      path: { type: 'string', required: true, description: 'Explicit destination such as daily/20260814.md. A filename is required.' },
      content: { type: 'string', required: true, description: 'The complete Markdown or text content for the new note.' },
    },
    'create_note',
  )

  register(
    'update_library_note',
    'Immediately save a new full version of an existing readable Markdown or text note. First call read_library_note and pass its item id and exact current_version. The host rejects a stale version rather than overwriting newer user changes. Only say the note was saved after this tool succeeds.',
    {
      item_id: { type: 'number', required: true, description: 'The note id returned by read_library_note.' },
      base_version: { type: 'number', required: true, description: 'The exact current_version returned by read_library_note.' },
      content: { type: 'string', required: true, description: 'The complete Markdown or text content to save as the next version.' },
    },
    'update_note',
  )
}
