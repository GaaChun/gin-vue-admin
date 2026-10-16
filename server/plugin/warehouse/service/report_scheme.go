package service

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

type SchemeAccess struct {
	CanQuery          bool
	CanManagePersonal bool
	CanManageShared   bool
}

// prepareScheme validates configuration before any persistent write.
func prepareScheme(in req.ReportSchemeInput, access SchemeAccess) (string, string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return "", "", domainError(ValidationFailed, "方案名称不能为空且最多100字", "name")
	}
	columns, err := validateColumns(in.Columns)
	if err != nil {
		return "", "", err
	}
	if in.Scope != "personal" && in.Scope != "shared" {
		return "", "", domainError(ValidationFailed, "方案范围无效", "scope")
	}
	if in.Scope == "shared" && !access.CanManageShared {
		return "", "", domainError(Forbidden, "缺少共享方案管理权限", "scope")
	}
	if in.Scope == "personal" && !access.CanManagePersonal {
		return "", "", domainError(Forbidden, "缺少个人方案维护权限", "scope")
	}
	encoded, err := json.Marshal(columnIDs(columns))
	return name, string(encoded), err
}
