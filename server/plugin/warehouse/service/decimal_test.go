package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseDecimalBoundaries(t *testing.T) {
	if _, err := ParseDecimal("99999999999999.9999", QuantityRule, "quantity"); err != nil { t.Fatalf("maximum quantity rejected: %v", err) }
	for _, input := range []string{"0", "1.00000", "100000000000000", "1e3", "-1", "NaN"} {
		if _, err := ParseDecimal(input, QuantityRule, "quantity"); err == nil { t.Fatalf("invalid quantity %q accepted", input) }
	}
	if got, err := ParseDecimal("0", PriceRule, "unitPrice"); err != nil || !got.IsZero() { t.Fatalf("zero price must be accepted: %v", err) }
}

func TestLineAmountsRoundEachLineBeforeSumming(t *testing.T) {
	quantity := decimal.RequireFromString("0.3333")
	price := decimal.RequireFromString("0.05")
	factor := decimal.RequireFromString("1")
	first, err := CalculateLine(quantity, price, factor)
	if err != nil { t.Fatal(err) }
	second, err := CalculateLine(quantity, price, factor)
	if err != nil { t.Fatal(err) }
	if got := FormatDecimal(SumAmounts([]LineAmounts{first, second}), 2); got != "0.04" { t.Fatalf("two rounded lines totalled %s, want 0.04", got) }
	if got := FormatDecimal(first.BaseQuantity, 10); got != "0.3333000000" { t.Fatalf("base quantity %s", got) }
}
