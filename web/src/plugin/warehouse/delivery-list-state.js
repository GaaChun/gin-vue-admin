import { routeTabKey } from '../../utils/route-state.js'

const states = new Map()

function stateKey(account, route) {
  const query = { ...route.query }
  delete query.whDelivery
  delete query.whId
  delete query.whNew
  return `${account}:${routeTabKey({ ...route, query })}`
}

export function readDeliveryListState(account, route) {
  const state = states.get(stateKey(account, route))
  return state ? structuredClone(state) : null
}

export function writeDeliveryListState(account, route, state) {
  states.set(stateKey(account, route), JSON.parse(JSON.stringify(state)))
}

export function clearDeliveryListState(account, route) {
  states.delete(stateKey(account, route))
}
