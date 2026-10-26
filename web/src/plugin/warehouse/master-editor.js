/** Server records are authoritative for the saved identity, version and audit data. */
export function masterEditorValues(detail) {
  return {
    id: detail.ID ?? detail.id,
    version: detail.version,
    name: detail.name || '',
    code: detail.code || '',
    mnemonic: detail.mnemonic || '',
    parentId: detail.parentId || 0,
    brandId: detail.brandId,
    categoryId: detail.categoryId,
    specification: detail.specification || '',
    alias: detail.alias || '',
    imageFileId: detail.imageFileId || null,
    units: (detail.units || []).map(unit => ({ rowKey: `saved-${unit.unitId}`, unitId: unit.unitId, factor: String(unit.factor), isBase: unit.isBase, enabled: unit.enabled }))
  }
}
