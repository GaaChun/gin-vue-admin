import assert from 'node:assert/strict'
import test from 'node:test'
import { accountIdentity, isCreatedEditorRoute, routeTabKey, savedMasterRoute, snapshotRoute } from './route-state.js'

const list = { name: 'warehouseMaterial', query: {}, params: {}, meta: {} }
const draft = { ...list, query: { whDetail: 'materials', whNew: '1' } }
const existing = id => ({ ...list, query: { whDetail: 'materials', whId: String(id) } })

test('list, draft, resources and each document have distinct cache identities', () => {
  const keys = [list, draft, existing(1), existing(2), { ...list, name: 'warehouseBrand' }].map(routeTabKey)
  assert.equal(new Set(keys).size, keys.length)
  assert.equal(routeTabKey(existing(1)), routeTabKey({ ...existing(1), query: { whId: '1', whDetail: 'materials' } }))
  assert.notEqual(routeTabKey(list), routeTabKey({ ...list, hash: '#details' }))
  assert.notEqual(accountIdentity({ ID: 1 }), accountIdentity({ ID: 2 }))
})

test('inactive page snapshots do not observe a shared live route changing tabs', () => {
  const route = { ...draft, query: { ...draft.query, tag: ['first'] }, params: { ids: ['1'] }, meta: { title: '新建物料' }, matched: [] }
  const own = snapshotRoute(route)
  route.query.whNew = undefined
  route.query.whId = '2'
  route.query.tag.push('second')
  route.params.ids.push('2')
  route.name = 'warehouseBrand'
  route.meta.title = '客户'
  route.matched.push({ path: '/brands' })
  assert.equal(own.name, 'warehouseMaterial')
  assert.equal(own.query.whNew, '1')
  assert.equal(own.query.whId, undefined)
  assert.deepEqual(own.query.tag, ['first'])
  assert.deepEqual(own.params.ids, ['1'])
  assert.equal(own.meta.title, '新建物料')
  assert.deepEqual(own.matched, [])
})

test('saving a draft switches only that draft to the server ID while keeping its context', () => {
  const source = { ...draft, query: { ...draft.query, whParent: '12', keyword: 'kept' }, params: { section: 'base' } }
  const saved = savedMasterRoute(source, 'materials', 42)
  assert.deepEqual(saved, { name: 'warehouseMaterial', params: { section: 'base' }, query: { whDetail: 'materials', whId: '42', keyword: 'kept' }, hash: '' })
  assert.equal(isCreatedEditorRoute(source, saved), true)
  assert.equal(isCreatedEditorRoute(list, saved), false)
  assert.equal(isCreatedEditorRoute(existing(1), saved), false)
  assert.equal(isCreatedEditorRoute(source, { ...saved, name: 'warehouseBrand' }), false)
  assert.equal(isCreatedEditorRoute(source, { ...saved, query: { ...saved.query, whDetail: 'brands' } }), false)
  assert.equal(isCreatedEditorRoute(source, source), false)
  assert.equal(source.query.whNew, '1')
})

test('announcement form draft replacement does not replace its list or another existing record', () => {
  const source = { name: 'announcementForm', query: {}, params: {}, meta: { path: '/src/plugin/announcement/form/info.vue' } }
  const saved = { ...source, query: { id: '18' } }
  assert.equal(isCreatedEditorRoute(source, saved), true)
  assert.equal(isCreatedEditorRoute({ ...source, query: { id: '17' } }, saved), false)
  assert.equal(isCreatedEditorRoute({ ...source, meta: { path: '/src/plugin/announcement/view/info.vue' } }, saved), false)
})
