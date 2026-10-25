import { download, request } from './common'

const base = '/warehouse/master'

/** 查询资料分页，筛选字段与后端各资源 DTO 一致。 */
export const listMaster = (kind, params) => request({ url: `${base}/${kind}/list`, method: 'get', params })
export const detailMaster = (kind, id) => request({ url: `${base}/${kind}/detail`, method: 'get', params: { id } })
export const saveMaster = (kind, data) => request({ url: `${base}/${kind}/${data.id ? 'update' : 'create'}`, method: data.id ? 'put' : 'post', data })
export const deleteMaster = (kind, data) => request({ url: `${base}/${kind}/delete`, method: 'delete', data })
export const getCategoryTree = () => request({ url: `${base}/categories/tree`, method: 'get' })
export const suggestMasterMnemonic = (name) => request({ url: `${base}/mnemonic`, method: 'post', data: { name }, donNotShowLoading: true })
export const reserveMasterCode = (resource) => request({ url: `${base}/reserveCode`, method: 'post', data: { resource }, donNotShowLoading: true })
export const uploadMasterImage = (file) => {
  const data = new FormData()
  data.append('file', file)
  return request({ url: `${base}/materials/uploadImage`, method: 'post', data })
}
export const importMaster = (kind, file) => {
  const data = new FormData()
  data.append('file', file)
  return request({ url: `${base}/${kind}/import`, method: 'post', data })
}
export const downloadMasterTemplate = (kind) => download({ url: `${base}/${kind}/template`, method: 'get' }, `${kind}-template.xlsx`)
/** 导出完整筛选集，不传页面分页。 */
export const exportMaster = (kind, params) => download({ url: `${base}/${kind}/export`, method: 'get', params }, `${kind}.xlsx`)
