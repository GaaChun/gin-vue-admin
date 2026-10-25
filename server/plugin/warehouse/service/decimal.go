package service

import (
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var decimalInput = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

type DecimalRule struct {
	Scale        int32
	MaxInteger   int
	AllowZero    bool
	OutputScale  int32
}

var (
	QuantityRule = DecimalRule{Scale: 4, MaxInteger: 14, OutputScale: 4}
	PriceRule    = DecimalRule{Scale: 4, MaxInteger: 14, AllowZero: true, OutputScale: 4}
	AmountRule   = DecimalRule{Scale: 2, MaxInteger: 36, AllowZero: true, OutputScale: 2}
	FactorRule   = DecimalRule{Scale: 6, MaxInteger: 12, OutputScale: 6}
)

func ParseDecimal(input string, rule DecimalRule, field string) (decimal.Decimal, error) {
	input = strings.TrimSpace(input)
	if !decimalInput.MatchString(input) { return decimal.Zero, domainError(ValidationFailed, "必须为非负普通十进制数", field) }
	parts := strings.SplitN(input, ".", 2)
	if len(parts[0]) > rule.MaxInteger { return decimal.Zero, domainError(ValidationFailed, "整数位超出范围", field) }
	if len(parts) == 2 && len(parts[1]) > int(rule.Scale) { return decimal.Zero, domainError(ValidationFailed, "小数位超出范围", field) }
	value, err := decimal.NewFromString(input)
	if err != nil { return decimal.Zero, domainError(ValidationFailed, "十进制数无效", field) }
	if !rule.AllowZero && !value.GreaterThan(decimal.Zero) { return decimal.Zero, domainError(ValidationFailed, "必须大于0", field) }
	return value, nil
}

func FormatDecimal(value decimal.Decimal, scale int32) string { return value.StringFixed(scale) }

type LineAmounts struct {
	BaseQuantity decimal.Decimal
	Amount       decimal.Decimal
}

func CalculateLine(quantity, unitPrice, factor decimal.Decimal) (LineAmounts, error) {
	if !quantity.GreaterThan(decimal.Zero) { return LineAmounts{}, domainError(ValidationFailed, "数量必须大于0", "quantity") }
	if unitPrice.IsNegative() { return LineAmounts{}, domainError(ValidationFailed, "单价不能为负数", "unitPrice") }
	if !factor.GreaterThan(decimal.Zero) { return LineAmounts{}, domainError(ValidationFailed, "换算系数必须大于0", "factor") }
	return LineAmounts{
		BaseQuantity: quantity.Mul(factor).Round(10),
		Amount: quantity.Mul(unitPrice).Round(2), // Monetary lines round before aggregation by contract.
	}, nil
}

// ResolveLinePrice preserves an entered amount even when its four-place
// derived unit price cannot multiply back to that same amount.
func ResolveLinePrice(quantity decimal.Decimal, unitPrice string, amount *string) (decimal.Decimal, *decimal.Decimal, error) {
	if amount == nil {
		price, err := ParseDecimal(unitPrice, PriceRule, "unitPrice")
		return price, nil, err
	}
	value, err := ParseDecimal(*amount, AmountRule, "amount")
	if err != nil {
		return decimal.Zero, nil, err
	}
	price := value.DivRound(quantity, 4)
	if _, err := ParseDecimal(price.StringFixed(4), PriceRule, "unitPrice"); err != nil {
		return decimal.Zero, nil, err
	}
	return price, &value, nil
}

func SumAmounts(lines []LineAmounts) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines { total = total.Add(line.Amount) }
	return total.Round(2)
}
