// Mnemonics stay a string in the API; each delimited alias is independently searchable.
export function matchesMasterKeyword(record, query, fields = ['name', 'code']) {
  const keyword = String(query ?? '').trim().toLocaleLowerCase()
  if (!keyword) return true
  return fields.some(field => String(record[field] ?? '').toLocaleLowerCase().includes(keyword)) ||
    String(record.mnemonic ?? '').split(/[,，;；\s]+/).some(alias => alias.toLocaleLowerCase().includes(keyword))
}

export function materialSelectionSummary(ids, materials) {
  if (!ids.length) return ''
  if (ids.length === 1) {
    const material = materials.find(item => (item.ID ?? item.id) === ids[0])
    if (material) return material.name
  }
  return `已选 ${ids.length} 种物料`
}
