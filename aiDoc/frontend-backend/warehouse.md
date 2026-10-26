# Warehouse 前后端契约（部分已落地）

> 单位、品牌、类别及助记符建议已有业务 HTTP 端点和 Swagger；物料、送货单、对账、方案、图片和所有 Excel 端点仍为 planned。本文中的 planned 路径不是可调用接口，实际 API 以已注册路由与 Swagger 为准。

## 通用响应与分页

实际端点落地后，成功响应统一为：

```json
{"code": 0, "data": {}, "msg": "操作成功"}
```

列表的 `data` 统一包含：

```json
{"page": 1, "pageSize": 20, "total": 0, "list": []}
```

计划默认 `page=1`、`pageSize=20`，允许范围为 `1..100`。详情无记录时返回业务错误，不以空对象模拟成功。报表查询在该分页结构外还会返回列定义、全局汇总和查询说明；全局汇总来自完整筛选集，不能由当前页累加。

## 写请求与服务端拥有字段

写 DTO 只接受每个资源明确列出的可编辑字段。客户端不得提交或信任以下服务端拥有字段：

- `id` 以外的审计字段：`createdAt`、`updatedAt`、`deletedAt`、`createdBy`、`updatedBy`、`reviewedBy`、`reviewedAt`；
- 送货单号、状态、明细 `lineNo`；
- 品牌、物料、类别、单位、基本单位、创建人和审核人的快照字段；
- 服务端计算字段：`baseQuantity`、行 `amount`、`totalAmount`；
- 任何由后端生成、校验或从当前资料读取的身份、快照、审核与金额字段。

后端从认证上下文提取操作者后传入 Service；Service 不依赖 `gin.Context`。写入时使用明确更新白名单，不以客户端完整模型或全量 `Save` 覆盖记录。

## 十进制字符串

所有数量、单价、系数、基本数量、金额和汇总值在 JSON 中都是字符串，不能是 JSON number：

```json
{
  "quantity": "2.0000",
  "unitPrice": "30.0000",
  "factor": "12.000000",
  "baseQuantity": "24.0000000000",
  "amount": "60.00"
}
```

计划中的校验和输出尺度如下：

| 字段 | 写入约束 | 输出格式 |
| --- | --- | --- |
| `quantity` | 必填、`>0`、最多 4 位小数 | 固定 4 位 |
| `unitPrice` | 必填、`>=0`、最多 4 位小数 | 固定 4 位 |
| `factor` | `>0`、最多 6 位小数 | 固定 6 位 |
| `baseQuantity` | 服务端以数量 × 系数计算 | 固定 10 位 |
| `amount`、`totalAmount` | 服务端逐行 half-up 到 2 位后汇总 | 固定 2 位 |

拒绝指数表示、NaN、Infinity、超精度、超字段范围及不允许的负数。前端不得用 `Number` 计算账务结果。送货单录入接受省略整数零或末尾小数点的写法，移焦后规范化为十进制字符串；清空数量、单价或金额时依据其余字段补算或填 `0`，数量、单价和金额均允许为 `0`。

## 时间、业务日与版本

接口时间使用带时区的 RFC3339；后端持久化统一转换 UTC。送货日、编号日期和按日/月报表维度按 `Asia/Shanghai` 业务时区计算，不依赖服务器 Local 时区。日期区间使用半开区间 `[from,to)`；选择结束日期时，前端应转换为下一日的 `00:00`，不能拼接 `23:59:59`。

可更新资源的 `version` 是可选正整数：

- 省略时跳过过期版本比较，但仍执行权限、事务、状态和引用校验；
- 提供时必须等于当前版本，显式 `0` 非法；
- 标准页面读取详情后应始终回传读取到的 `version`；
- 修改、删除、审核和反审核成功后，主记录版本只递增一次。

## 基础资料字段、查重与类别

资料写入按 **UTF-8 字符数** 限制字符串，不能依赖数据库截断：单位 `name` 最多 100；品牌和类别的 `name`、`mnemonic` 最多 100、`code` 最多 64；物料 `name` 最多 200、`specification` 最多 255、`alias` 最多 200，物料 `mnemonic` 最多 100、`code` 最多 64。

活跃记录的唯一范围为：单位 `name`；品牌 `name`、`code`；类别 `code`、`(parentId,name)`；物料 `code`；物料单位 `(materialId,unitId)`；个人方案 `(ownerId,name)`；共享方案 `name`。比较会去除首尾空白并按大小写不敏感处理。仅未软删除记录参与查重，软删除资料允许复用编号或名称；历史引用必须继续使用稳定 ID 和快照，不能按复用后的编号重新关联。送货单号、编号日期和预占 token 全历史唯一，不适用软删复用。

类别 `parentId=0` 表示虚拟根，不是实际类别 ID。真实类别的 `code` 不得为保留值 `ROOT`；**planned** Excel 导入中，父级为根以 `ROOT` 表示。类别树应一次读取有效节点后在内存组装；用户选定父类别筛选时，应包含该类别及全部后代，未传父筛选时仍保持未筛选查询。

## 已落地基础资料接口

以下路径在框架 `RouterPrefix` 后挂载于 `/warehouse`，已注册业务处理函数；均使用本文的统一响应和分页约定，详情缺失返回业务错误。它们不包含 Excel 模板、导入或导出能力。

| 资源 | 已落地方法与路径 | 输入要点 |
| --- | --- | --- |
| 单位 | `GET /warehouse/master/units/list`、`GET /warehouse/master/units/detail?id=`、`POST /warehouse/master/units/create`、`PUT /warehouse/master/units/update`、`DELETE /warehouse/master/units/delete` | `name` 为唯一可编辑业务字段；更新、删除接受 `{id,version?}`。 |
| 品牌 | `GET /warehouse/master/brands/list`、`GET /warehouse/master/brands/detail?id=`、`POST /warehouse/master/brands/create`、`PUT /warehouse/master/brands/update`、`DELETE /warehouse/master/brands/delete` | 写入 `{name,mnemonic,code}`；更新、删除接受 `{id,version?}`。 |
| 类别 | `GET /warehouse/master/categories/list`、`GET /warehouse/master/categories/detail?id=`、`GET /warehouse/master/categories/tree`、`POST /warehouse/master/categories/create`、`PUT /warehouse/master/categories/update`、`DELETE /warehouse/master/categories/delete` | 写入 `{name,mnemonic,code,parentId}`，其中根为 `parentId:0`；更新、删除接受 `{id,version?}`。 |
| 助记符建议 | `POST /warehouse/master/mnemonic` | `{name}` 返回 `{mnemonic}`；只供相应资料的新建/编辑权限使用。 |
| 部门 | `GET /warehouse/master/departments/list`、`GET /warehouse/master/departments/detail?id=`、`POST /warehouse/master/departments/create`、`PUT /warehouse/master/departments/update`、`DELETE /warehouse/master/departments/delete` | 写入 `{name,mnemonic}`，名称必填且最多 100 字符，助记符可空且最多 100 字符；更新、删除接受 `{id,version?}`。列表 `keyword` 匹配名称或助记符。 |

## 资源契约与实现状态

以下路径均位于框架 `RouterPrefix` 后的 `/warehouse` 下；仅明确标为 planned 的接口尚未实现。

| 资源 | 路径与方法 | 请求/响应要点 |
| --- | --- | --- |
| 物料 | `GET /master/materials/list`、`/detail?id=`；`POST /create`、`PUT /update`、`DELETE /delete` | 已落地。创建/更新提交完整单位集合 `{unitId,factor,isBase,enabled}`；更新按 `unitId` 保留绑定 ID。允许重新指定基本单位，始终要求唯一启用基本单位且系数为 `1`；省略被引用绑定仍会拒绝，未被引用的绑定随完整集合软删。变更不改写历史送货明细的基本单位、系数和基本数量快照。 |
| 资料 Excel | `GET /master/{units,brands,categories,materials}/template`、`POST .../import`、`GET .../export` | 已落地。单位按名称，品牌/类别/物料按编号新增或更新；未知品牌、类别、单位不会自动创建。类别可在同文件前向引用父级，父级为虚拟根时填保留值 `ROOT`。物料必须同时提供 `物料单位` 完整集合，省略绑定表示删除而不是保留。空白可选列清空旧值。任一行失败整文件回滚。 |
| 物料图片 | `POST /master/materials/uploadImage` | 已落地。multipart `file`，仅 jpg/jpeg/png/bmp，最大 1MiB，不自动压缩；成功返回 `{fileId,url,size,mime}`。物料写请求只接受该入口登记过的 `fileId`。 |
| 送货单 | `POST /delivery/reserveNumber`；`GET /delivery/list`、`/detail?id=`；`POST /create`、`PUT /update`、`DELETE /delete`；`POST /review`、`/unreview` | 已落地。预占返回正式号和 token，不创建空单；创建消费 token。更新按明细 ID 保留快照，跨品牌或已审核均整单拒绝。 |
| 单据动作 | `POST /delivery/reserveNumber`、`/review`、`/unreview` | 审核与反审核接收 `{id,version?}`；预占单号仅由后端生成。 |
| 录单候选 | 当前使用 `/master/brands/list`、`/master/categories/list`、`/master/units/list`、`/master/materials/list` 与 `/master/materials/detail` | 录单角色须有这些只读权限；候选仅按当前表头品牌展示。专用 `/delivery/materialOptions`、`/brandOptions` 未实现。 |
| 送货单 Excel | `GET /delivery/template`、`POST /delivery/import`、`GET /delivery/export` | planned，尚未实现，不在页面显示伪操作入口。 |
| 对账 | `GET /reconciliation/columns`、`/options`；`POST /reconciliation/query`、`/export`、`/printData` | 已接入数据库已审核快照。查询分页，导出 xlsx，打印返回全量 `{projection,filterDescription,summaryLines,title}`。 |
| 方案 | `GET /schemes/list`、`/detail?id=`；`POST /schemes/create`、`/publish`；`PUT /schemes/update`、`DELETE /schemes/delete` | 持久化个人/共享方案，仅保存有序列集合，不保存筛选。共享增改删及范围变更额外校验 `POST /warehouse/schemes/publish` 权限。列表返回 `permissions:{create,update,delete,shared}`。 |

创建送货单 DTO（详情/保存当前返回 `{Note,Details}`）：

```json
{
  "reservationToken": "uuid-token",
  "deliveredAt": "2026-10-15T09:30:00+08:00",
  "brandId": 1,
  "details": [
    {"materialId": 1, "materialUnitId": 2, "quantity": "2.0000", "unitPrice": "30.0000"}
  ]
}
```

更新送货单 DTO 额外包含 `id`、可选 `version`，每条明细用可选 `id` 标识已有行。`lineNo` 由后端按数组顺序生成；写请求不接收单号、快照、计算值、状态或审核字段。

创建物料 DTO 为 `{name,mnemonic,code,brandId,categoryId,specification?,alias?,imageFileId?,units:[{unitId,factor,isBase,enabled}]}`；更新再加 `{id,version?}`。详情返回完整单位绑定及只读 `imageUrl`。资料导出携带与列表相同的 `name/keyword/brandId/categoryId/parentId` 筛选，忽略分页。

物料、客户和部门继续使用 `mnemonic` 字符串，允许多个分隔的助记符；名称、编号及任一助记符均可按各资料检索规则命中，不要求迁移旧的单个助记符。

部门资料为独立 `wh_departments` 表；启动迁移将未删除送货单的历史部门名称一次性导入，完成标记保存在 `wh_department_imports`，避免重启后复活已改名或已删除的部门。送货单保存时在同一事务中登记自由输入的新部门。资料改名、删除不改写历史送货单快照；原 `/delivery/departments` 仍返回历史名称字符串数组，保持报表筛选语义。

### 对账投影

`POST /reconciliation/query` 接受 `{filters:{deliveredFrom?,deliveredTo?,reviewedFrom?,reviewedTo?,number?,brandId?,categoryId?,materialKeyword?},columns:[...],page,pageSize}`。时间为半开区间；品牌/类别与物料检索读取历史快照，类别精确匹配，不含当前树的后代。

列字典以 `GET /reconciliation/columns` 为准；所有 `dimension` 列决定分组，数量/基本数量只在物料与单位可比时相加，单价只显示组内一致值。D8 样例金额为 `68.50`，混合箱/件数量为 null、基本数量为 `28.0000000000`。响应包含 `{list,total,page,pageSize,columns,summary,warnings}`；行使用 `values`、`unavailable`、`baseUnits`。合计来自全部筛选行，不取当前页；隐藏指标同时隐藏其合计。不同请求实时读取数据，不保证查询与稍后导出之间冻结。

### 基础资料 Excel 示例

合法物料文件：

| 工作表 | 示例 |
| --- | --- |
| `物料` | `M001,普通文本,wb,规格,,B001,C001,,1` |
| `物料单位` | `M001,个,1,是,是`；`M001,箱,12.5,否,是` |

失败物料文件：`物料单位` 出现主表不存在的物料编号，或省略仍被送货明细引用的绑定。整份文件不提交，已解析的其他合法行也不保留。

导入顺序固定为单位→品牌→类别→物料。空白规格/别名/图片文件 ID 清空旧值；版本号空白表示不校验，`0` 非法。

## 错误响应

业务失败保留框架的 `code != 0` 与中文 `msg`，并在 `data` 中统一补充：

```json
{
  "errorCode": "VERSION_CONFLICT",
  "field": "version",
  "lineNo": 12,
  "currentVersion": 3,
  "errors": [{"sheet": "送货单", "row": 12, "column": "单价", "reason": "最多4位小数"}]
}
```

`field`、`lineNo`、`currentVersion` 与 `errors` 按错误类型可省略。计划中的 `errorCode` 枚举为：`VALIDATION_FAILED`、`DUPLICATE_VALUE`、`VERSION_CONFLICT`、`STATE_CONFLICT`、`REFERENCE_IN_USE`、`NOT_FOUND`、`FORBIDDEN`、`IMPORT_FAILED`、`NUMBER_EXHAUSTED`。不得向前端暴露数据库原始错误或 SQL。

导出成功时才返回具有正确 MIME 和 `Content-Disposition` 的 xlsx 流；失败仍返回上述标准 JSON，下载工具必须识别 JSON 错误而非保存为 xlsx。