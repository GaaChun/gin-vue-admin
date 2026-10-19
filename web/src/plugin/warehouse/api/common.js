import service from '@/utils/request'
import { ElMessage } from 'element-plus'

export const idOf = (row) => row?.ID ?? row?.id

export function pageSequenceNumber(index, page, pageSize) {
  return (page - 1) * pageSize + index + 1
}

export function unwrap(response) {
  if (response?.code === 0) return response.data
  const payload = response?.data?.code !== undefined ? response.data : response
  const error = new Error(payload?.msg || payload?.data?.message || '请求失败')
  error.business = payload?.data
  throw error
}

export async function request(config) {
  const headers = config.data instanceof FormData ? { 'Content-Type': 'multipart/form-data', ...config.headers } : config.headers
  return unwrap(await service({ ...config, headers }))
}

export function showError(error) {
  if (error === 'cancel' || error === 'close') return
  const business = error?.business || error?.response?.data?.data
  const rows = business?.errors || []
  const detail = rows.map((row) => [row.sheet, row.lineNo ? `第${row.lineNo}行` : '', row.column || row.field, row.message].filter(Boolean).join('：')).join('\n')
  ElMessage({ type: 'error', showClose: true, duration: detail ? 10000 : 5000, message: detail || business?.message || error?.message || '操作失败' })
}

export async function download(config, filename) {
  const response = await service({ ...config, responseType: 'blob' })
  if (response?.code !== undefined) return unwrap(response)
  const blob = response.data
  if (!(blob instanceof Blob)) throw new Error('下载响应无效')
  if (blob.type.includes('json') || (await blob.slice(0, 1).text()) === '{') {
    unwrap(JSON.parse(await blob.text()))
    throw new Error('服务器未返回文件')
  }
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export async function allOptions(kind) {
  const list = []
  let page = 1
  while (true) {
    const data = await request({ url: `/warehouse/master/${kind}/list`, method: 'get', params: { page, pageSize: 100 } })
    list.push(...(data.list || []))
    if (list.length >= data.total || !data.list?.length) return list
    page++
  }
}
