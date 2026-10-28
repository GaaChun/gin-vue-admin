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
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatal(err)
	}
	if len(input.Details) != 1 || input.Details[0].Remark != "  明细备注  " {
		t.Fatalf("remark input=%#v", input)
	}
	var query DeliveryListQuery
	request := httptest.NewRequest("GET", "/warehouse/delivery/list?detailRemark=needle+text&page=2&pageSize=1", nil)
	if err := binding.Query.Bind(request, &query); err != nil {
		t.Fatal(err)
	}
	if query.DetailRemark != "needle text" || query.Page != 2 || query.PageSize != 1 {
		t.Fatalf("query=%#v", query)
	}
}

func TestDeliveryAuditFilterBindings(t *testing.T) {
	var query DeliveryListQuery
	request := httptest.NewRequest("GET", "/warehouse/delivery/list?createdBy=%E5%88%9B%E5%BB%BA%E4%BA%BA&updatedBy=%E4%BF%AE%E6%94%B9%E4%BA%BA&reviewedBy=%E5%AE%A1%E6%A0%B8%E4%BA%BA&createdFrom=2026-10-01T00:00:00%2B08:00&createdTo=2026-10-02T00:00:00%2B08:00&updatedFrom=2026-10-03T00:00:00%2B08:00&updatedTo=2026-10-04T00:00:00%2B08:00&reviewedFrom=2026-10-05T00:00:00%2B08:00&reviewedTo=2026-10-06T00:00:00%2B08:00", nil)
	if err := binding.Query.Bind(request, &query); err != nil {
		t.Fatal(err)
	}
	if query.CreatedBy != "创建人" || query.UpdatedBy != "修改人" || query.ReviewedBy != "审核人" {
		t.Fatalf("name filters=%#v", query)
	}
	if query.CreatedFrom == nil || query.CreatedTo == nil || query.UpdatedFrom == nil || query.UpdatedTo == nil || query.ReviewedFrom == nil || query.ReviewedTo == nil {
		t.Fatalf("time filters were not bound: %#v", query)
	}
}
