export const fmtTitle = (title, now) => {
  const reg = /\$\{(.+?)\}/
  const reg_g = /\$\{(.+?)\}/g
  const result = title.match(reg_g)
  if (result) {
    result.forEach((item) => {
      const key = item.match(reg)[1]
      const value = now.params[key] || now.query[key]
      title = title.replace(item, value)
    })
  }
  const query = now?.query || {}
  if (query.whDetail || query.whDelivery === '1') {
    title = `${title}-${query.whNew === '1' ? '新建' : '编辑'}`
  }
  return title
}
