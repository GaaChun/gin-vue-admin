package request

import "testing"

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
