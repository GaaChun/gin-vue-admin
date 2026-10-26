function routeValues(values = {}) {
  return Object.fromEntries(Object.keys(values).sort().filter(key => values[key] !== undefined).map(key => [key, Array.isArray(values[key]) ? [...values[key]] : values[key]]))
}

/** One identity for the tab bar and cache, independent of query key order. */
export function routeTabKey(route) {
  return JSON.stringify([route.name || route.path, routeValues(route.query), routeValues(route.params), route.hash || ''])
}

/** A cached page must never observe another tab's live useRoute object. */
export function snapshotRoute(route) {
  return { ...route, query: routeValues(route.query), params: routeValues(route.params), meta: { ...route.meta }, matched: [...(route.matched || [])] }
}

export function accountIdentity(user = {}) {
  return String(user.ID || user.id || user.uuid || '')
}

/** Saving a new document replaces its draft tab, not the list or another ID. */
export function isCreatedEditorRoute(previous, next) {
  if (!previous || previous.name !== next.name) return false
  const warehouse = previous.query?.whNew === '1' && next.query?.whId && !next.query?.whNew
  const announcement = previous.meta?.path?.endsWith('/announcement/form/info.vue') && !previous.query?.id && next.query?.id
  if (!warehouse && !announcement) return false
  const withoutDocument = (route) => {
    const query = { ...route.query }
    delete query.whId
    delete query.whNew
    delete query.whParent
    if (announcement) delete query.id
    return routeTabKey({ ...route, query })
  }
  return withoutDocument(previous) === withoutDocument(next)
}

export function savedMasterRoute(route, resource, id) {
  const query = { ...route.query, whDetail: resource, whId: String(id) }
  delete query.whNew
  delete query.whParent
  return { name: route.name, params: { ...route.params }, query, hash: route.hash || '' }
}
