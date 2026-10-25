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

func TestEnteredAmountIsAuthoritativeAndDerivedPriceIsBounded(t *testing.T) {
	for _, tc := range []struct { quantity, amount, price string }{
		{"3", "10.00", "3.3333"},
		{"10000", "9.99", "0.0010"},
		{"2", "0.01", "0.0050"},
		{"2", "0", "0.0000"},
	} {
		quantity := decimal.RequireFromString(tc.quantity)
		price, amount, err := ResolveLinePrice(quantity, "ignored", &tc.amount)
		if err != nil || price.StringFixed(4) != tc.price || amount == nil || amount.StringFixed(2) != decimal.RequireFromString(tc.amount).StringFixed(2) {
			t.Fatalf("%#v price=%s amount=%v err=%v", tc, price, amount, err)
		}
	}
	for _, input := range []string{"-1", "1.001", "1e3", "NaN", "1000000000000000000000000000000000000"} {
		if _, _, err := ResolveLinePrice(decimal.NewFromInt(1), "1", &input); err == nil {
			t.Fatalf("invalid amount %q accepted", input)
		}
	}
	overflow := "10000000000"
	if _, _, err := ResolveLinePrice(decimal.RequireFromString("0.0001"), "1", &overflow); err == nil {
		t.Fatal("out-of-range derived price accepted")
	}
	price, amount, err := ResolveLinePrice(decimal.NewFromInt(3), "2.5", nil)
	if err != nil || price.StringFixed(4) != "2.5000" || amount != nil {
		t.Fatalf("ordinary price changed: %s %v %v", price, amount, err)
	}
}
