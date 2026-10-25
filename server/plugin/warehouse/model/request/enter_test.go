package request

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestPageQueryWithDefaults(t *testing.T) {
	defaults := (PageQuery{}).WithDefaults()
	if defaults.Page != 1 || defaults.PageSize != 20 {
		t.Fatalf("defaults = %#v", defaults)
	}

	explicit := (PageQuery{Page: 3, PageSize: 50}).WithDefaults()
	if explicit.Page != 3 || explicit.PageSize != 50 {
		t.Fatalf("explicit page was changed: %#v", explicit)
	}
}

func TestDeliveryRemarkBindings(t *testing.T) {
	var input CreateDeliveryInput
	body := `{"reservationToken":"reserved","brandId":1,"deliveredAt":"2026-10-25T00:00:00Z","details":[{"materialId":1,"materialUnitId":1,"quantity":"1","unitPrice":"2","remark":"  明细备注  "}]}`
	if err := json.Unmarshal([]byte(body), &input); err != nil { t.Fatal(err) }
	if len(input.Details) != 1 || input.Details[0].Remark != "  明细备注  " { t.Fatalf("remark input=%#v", input) }
	var query DeliveryListQuery
	request := httptest.NewRequest("GET", "/warehouse/delivery/list?detailRemark=needle+text&page=2&pageSize=1", nil)
	if err := binding.Query.Bind(request, &query); err != nil { t.Fatal(err) }
	if query.DetailRemark != "needle text" || query.Page != 2 || query.PageSize != 1 { t.Fatalf("query=%#v", query) }
}
