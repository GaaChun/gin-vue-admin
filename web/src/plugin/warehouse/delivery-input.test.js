import { describe, expect, it } from 'vitest'
import { calculateAmount, decimalError, deliveryCellTarget, deliveryForwardTarget, settleDeliveryNumber, trimDecimal, updateDeliveryNumber } from './delivery-input'

describe('delivery decimal entry', () => {
  it.each([['.5', '0.5'], ['4.', '4'], ['.05', '0.05'], ['982.', '982']])('normalizes %s only at the commit boundary', (input, expected) => {
    const row = { quantity: '2', unitPrice: input, amount: '0', amountEdited: false }
    const editing = updateDeliveryNumber(row, 'unitPrice')
    expect(editing.unitPrice).toBe(input)
    expect(settleDeliveryNumber(editing, 'unitPrice').unitPrice).toBe(expected)
    expect(trimDecimal(input)).toBe(expected)
    expect(decimalError(input, 'unitPrice')).toBe('')
  })

  it('restores a cleared amount from quantity and price without changing price', () => {
    const editing = updateDeliveryNumber({ quantity: '3', unitPrice: '0.335', amount: '', amountEdited: true }, 'amount')
    expect(editing.amount).toBe('')
    const committed = settleDeliveryNumber(editing, 'amount')
    expect(committed).toMatchObject({ unitPrice: '0.335', amount: '1.01', amountEdited: false })
    expect(decimalError(committed.amount, 'amount')).toBe('')
  })

  it.each(['quantity', 'unitPrice'])('fills cleared %s with zero and recomputes the amount', (field) => {
    const committed = settleDeliveryNumber({ quantity: '2', unitPrice: '3', amount: '6', [field]: '' }, field)
    expect(committed[field]).toBe('0')
    expect(committed.amount).toBe('0')
    expect(decimalError(committed[field], field)).toBe('')
  })

  it('keeps exact half-up rounding and amount override reverse calculation', () => {
    expect(calculateAmount('99999999999999.9999', '0.0001')).toBe('10000000000.00')
    expect(calculateAmount('.5', '.05')).toBe('0.03')
    const editing = updateDeliveryNumber({ quantity: '3', unitPrice: '0', amount: '10.', amountEdited: false }, 'amount')
    const override = settleDeliveryNumber(editing, 'amount')
    expect(override).toMatchObject({ unitPrice: '3.3333', amount: '10', amountEdited: true })
    expect(updateDeliveryNumber({ ...override, quantity: '4' }, 'quantity')).toMatchObject({ amount: '13.33', amountEdited: false })
  })

  it('does not turn unchanged calculated money into an override or recalculate saved override prices on blur', () => {
    expect(settleDeliveryNumber({ quantity: '3', unitPrice: '0.335', amount: '1.01', amountEdited: false }, 'amount')).toMatchObject({ unitPrice: '0.335', amountEdited: false })
    expect(settleDeliveryNumber({ quantity: '3', unitPrice: '3.3', amount: '10.00', amountEdited: true }, 'amount')).toMatchObject({ unitPrice: '3.3', amount: '10', amountEdited: true })
  })

  it.each(['-1', '1e2', 'Infinity', '.', '1.00001', '100000000000000'])('does not normalize invalid quantity %s into an accepted value', (input) => {
    const committed = settleDeliveryNumber({ quantity: input, unitPrice: '1', amount: '' }, 'quantity')
    expect(committed.quantity).toBe(input)
    expect(decimalError(committed.quantity, 'quantity')).not.toBe('')
  })

  it('retains amount precision and integer range checks', () => {
    expect(decimalError('0.001', 'amount')).not.toBe('')
    expect(decimalError('1000000000000000000', 'amount')).not.toBe('')
    expect(decimalError('999999999999999999.99', 'amount')).toBe('')
  })
})

describe('delivery keyboard targets', () => {
  const fields = ['material', 'unit', 'quantity', 'unitPrice', 'amount', 'remark']
  it('returns from quantity directly to material when the unit selector is disabled', () => {
    expect(deliveryCellTarget(fields, 'quantity', true, false)).toEqual({ field: 'material', rowOffset: 0 })
    expect(deliveryCellTarget(fields, 'quantity', true, true)).toEqual({ field: 'unit', rowOffset: 0 })
  })
  it('preserves cross-row movement and skips remarks', () => {
    expect(deliveryCellTarget(fields, 'material', true, false)).toEqual({ field: 'amount', rowOffset: -1 })
    expect(deliveryCellTarget(fields, 'amount', false, true)).toEqual({ field: 'material', rowOffset: 1 })
    expect(deliveryCellTarget(fields, 'remark', true, true)).toEqual({ field: 'amount', rowOffset: 0 })
  })
  it('uses the configured column order rather than assuming amount is last', () => {
    expect(deliveryCellTarget(['material', 'amount', 'remark', 'quantity', 'unitPrice'], 'unitPrice', false, true)).toEqual({ field: 'material', rowOffset: 1 })
  })
  it('appends once from a material row, then returns the blank tail to material without appending again', () => {
    const completedRow = deliveryForwardTarget(fields, 'amount', true, false)
    expect(completedRow).toEqual({ field: 'material', rowOffset: 1, append: true })
    const rows = [{ materialId: 7 }]
    if (completedRow.append) rows.push({ materialId: undefined })
    for (let attempt = 0; attempt < 3; attempt++) {
      const blankTail = deliveryForwardTarget(fields, 'amount', !!rows.at(-1).materialId, false)
      if (blankTail.append) rows.push({ materialId: undefined })
      expect(blankTail).toEqual({ field: 'material', rowOffset: 0, append: false })
    }
    expect(rows).toHaveLength(2)
  })
  it('uses the next displayed row instead of appending a duplicate', () => {
    expect(deliveryForwardTarget(fields, 'amount', true, true)).toEqual({ field: 'material', rowOffset: 1, append: false })
    expect(deliveryForwardTarget(fields, 'quantity', true, false)).toEqual({ field: 'unitPrice', rowOffset: 0, append: false })
  })
})
