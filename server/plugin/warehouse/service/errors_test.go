package service

import (
	"errors"
	"fmt"
	"testing"
)

func TestDomainErrorConvertsWorkbookDetailsAndSupportsErrorsAs(t *testing.T) {
	cellError := NewDomainError(ValidationFailed, "数量必须大于0").
		WithField("quantity").WithExcelLocation("送货单", 8, "数量")
	root := NewDomainError(ImportFailed, "导入失败").WithErrors(cellError)
	wrapped := fmt.Errorf("import command: %w", root)

	var extracted *DomainError
	if !errors.As(wrapped, &extracted) || extracted != root {
		t.Fatal("wrapped domain error must remain discoverable through errors.As")
	}
	if converted, ok := AsDomainError(wrapped); !ok || converted != root {
		t.Fatal("domain error helper must preserve the original domain error")
	}

	payload := root.BusinessError()
	if payload.ErrorCode != string(ImportFailed) || payload.Message != "导入失败" || len(payload.Errors) != 1 {
		t.Fatalf("unexpected root payload: %#v", payload)
	}
	item := payload.Errors[0]
	if item.ErrorCode != string(ValidationFailed) || item.Field != "quantity" ||
		item.Sheet != "送货单" || item.LineNo != 8 || item.Column != "数量" {
		t.Fatalf("workbook location was not preserved: %#v", item)
	}
}
