import { request, download } from './common'

const root = '/warehouse'
/** Fetch the server-owned column registry and historical filter candidates. */
export const loadReportColumns = () => request({ url: `${root}/reconciliation/columns`, method: 'get' })
export const loadReportOptions = () => request({ url: `${root}/reconciliation/options`, method: 'get' })
/** Query grouped rows, global exact summaries and the selected-column snapshot. */
export const queryReport = (data) => request({ url: `${root}/reconciliation/query`, method: 'post', data })
export const exportReport = (data) => download({ url: `${root}/reconciliation/export`, method: 'post', data }, '送货对账表.xlsx')
export const printReport = (data) => request({ url: `${root}/reconciliation/printData`, method: 'post', data })
export const listReportSchemes = (params) => request({ url: `${root}/schemes/list`, method: 'get', params })
export const getReportScheme = (id) => request({ url: `${root}/schemes/detail`, method: 'get', params: { id } })
export const createReportScheme = (data) => request({ url: `${root}/schemes/create`, method: 'post', data })
export const updateReportScheme = (data) => request({ url: `${root}/schemes/update`, method: 'put', data })
export const deleteReportScheme = (data) => request({ url: `${root}/schemes/delete`, method: 'delete', data })
export const publishReportScheme = (data) => request({ url: `${root}/schemes/publish`, method: 'post', data })
