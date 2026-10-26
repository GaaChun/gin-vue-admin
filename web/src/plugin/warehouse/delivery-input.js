const NUMBER_FIELDS = { quantity: { label: '数量', scale: 4, integerDigits: 14 }, unitPrice: { label: '单价', scale: 4, integerDigits: 14 }, amount: { label: '金额', scale: 2, integerDigits: 18 } }
const DECIMAL = /^(?:\d+(?:\.\d*)?|\.\d+)$/

export function trimDecimal(value) {
  const text = String(value ?? '').trim()
  if (!DECIMAL.test(text)) return text
  const [integer, fraction = ''] = text.split('.')
  const whole = integer.replace(/^0+(?=\d)/, '') || '0'
  const decimal = fraction.replace(/0+$/, '')
  return decimal ? `${whole}.${decimal}` : whole
}

export function parseScaled(value, scale) {
  const text = String(value ?? '').trim()
  if (!DECIMAL.test(text)) return null
  const [integer, fraction = ''] = text.split('.')
  if (fraction.length > scale) return null
  return BigInt(`${integer || '0'}${fraction.padEnd(scale, '0')}`)
}

export function formatScaled(value, scale) {
  const digits = value.toString().padStart(scale + 1, '0')
  return `${digits.slice(0, -scale)}.${digits.slice(-scale)}`
}

export function calculateAmount(quantityValue, priceValue) {
  const quantity = parseScaled(quantityValue, 4), price = parseScaled(priceValue, 4)
  if (quantity === null || price === null) return undefined
  return formatScaled((quantity * price + 500000n) / 1000000n, 2)
}

export function decimalError(value, field) {
  const text = String(value ?? '').trim()
  const { label, scale, integerDigits } = NUMBER_FIELDS[field]
  if (!text) return `请输入${label}`
  const [integer, fraction = ''] = text.split('.')
  if (!DECIMAL.test(text) || (integer.replace(/^0+/, '') || '0').length > integerDigits || fraction.length > scale) {
    return `${label}须为普通十进制数，整数最多${integerDigits}位、小数最多${scale}位`
  }
  return ''
}

// Keep the edited field verbatim until blur/navigation; dependent values can update live.
export function updateDeliveryNumber(row, field) {
  const result = { quantity: row.quantity, unitPrice: row.unitPrice, amount: row.amount, amountEdited: row.amountEdited }
  if (field === 'amount') {
    result.amountEdited = true
    const amount = parseScaled(row.amount, 2), quantity = parseScaled(row.quantity, 4)
    if (amount !== null && quantity !== null && quantity > 0n) {
      result.unitPrice = trimDecimal(formatScaled((amount * 1000000n + quantity / 2n) / quantity, 4))
    }
  } else {
    result.amountEdited = false
    const amount = calculateAmount(row.quantity, row.unitPrice)
    result.amount = amount == null ? amount : trimDecimal(amount)
  }
  return result
}

export function settleDeliveryNumber(row, field) {
  const value = String(row[field] ?? '').trim()
  if (value && decimalError(value, field)) return { quantity: row.quantity, unitPrice: row.unitPrice, amount: row.amount, amountEdited: row.amountEdited }
  if (field === 'amount' && !value) {
    const calculated = calculateAmount(row.quantity, row.unitPrice)
    if (calculated !== undefined) return { quantity: row.quantity, unitPrice: row.unitPrice, amount: trimDecimal(calculated), amountEdited: false }
  }
  if (value) return { quantity: row.quantity, unitPrice: row.unitPrice, amount: row.amount, amountEdited: row.amountEdited, [field]: trimDecimal(value) }
  return updateDeliveryNumber({ ...row, [field]: '0' }, field)
}

export function deliveryCellTarget(fields, field, backward, hasMaterial) {
  const index = fields.indexOf(field)
  const enabled = key => key !== 'remark' && (key !== 'unit' || hasMaterial)
  const target = backward ? fields.slice(0, index).reverse().find(enabled) : fields.slice(index + 1).find(enabled)
  if (target) return { field: target, rowOffset: 0 }
  const candidates = fields.filter(key => key !== 'remark')
  return { field: backward ? candidates.at(-1) : candidates[0], rowOffset: backward ? -1 : 1 }
}

export function deliveryForwardTarget(fields, field, hasMaterial, hasNextRow) {
  const target = deliveryCellTarget(fields, field, false, hasMaterial)
  if (target.rowOffset && !hasNextRow && !hasMaterial) return { field: 'material', rowOffset: 0, append: false }
  return { ...target, append: !!target.rowOffset && !hasNextRow }
}
