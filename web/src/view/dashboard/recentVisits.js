const MAX_RECENT_VISITS = 6

export function defaultMenuEntries(menus) {
  const entries = []
  const visit = (items) => {
    for (const item of items || []) {
      if (
        item.ID &&
        !item.hidden &&
        !item.children?.length &&
        item.name &&
        item.path &&
        !/^https?:\/\//i.test(item.path) &&
        item.name !== 'Reload'
      ) {
        entries.push({
          id: Number(item.ID),
          name: item.name,
          title: item.meta?.title || item.name
        })
      }
      visit(item.children)
    }
  }
  visit(menus)
  return entries.sort((a, b) => a.id - b.id).slice(0, MAX_RECENT_VISITS)
}

export function sortRecentVisits(items) {
  return [...items].sort((a, b) =>
    b.visitCount - a.visitCount ||
    Date.parse(b.lastVisitedAt) - Date.parse(a.lastVisitedAt) ||
    a.id - b.id
  ).slice(0, MAX_RECENT_VISITS)
}
