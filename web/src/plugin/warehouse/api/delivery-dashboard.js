import { request } from './common'

export const listDeliveryNotes = (params) => request({ url: '/warehouse/delivery/list', method: 'get', params })
