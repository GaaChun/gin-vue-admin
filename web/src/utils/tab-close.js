import { routeTabKey } from './route-state.js'

const handlers = new Map()

/** Guards belong to cached instances, including inactive editor tabs. */
export function registerTabClose(route, handler) {
  const key = routeTabKey(route)
  handlers.set(key, handler)
  return () => { if (handlers.get(key) === handler) handlers.delete(key) }
}

/** Ask every affected editor before changing any tab or discarding any draft. */
export async function prepareTabClose(tabs) {
  for (const tab of tabs) {
    const handler = handlers.get(routeTabKey(tab))
    if (handler?.beforeClose && !(await handler.beforeClose())) return false
  }
  return true
}

export function completeTabClose(tabs) {
  for (const tab of tabs) handlers.get(routeTabKey(tab))?.closed?.()
}
