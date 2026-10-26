# Warehouse 插件

> 状态：进行中。单位、品牌、类别及助记符建议已有 HTTP 接口；物料、送货单、对账和 Excel 等能力仍是计划，不能将 planned 路径当作可用功能。

## 职责与入口

`warehouse` 是物料基础资料、送货单及对账报表的业务插件。后端规划入口为 `server/plugin/warehouse/`，前端规划入口为 `web/src/plugin/warehouse/`；跨栈字段和资源契约见 [前后端契约](../frontend-backend/warehouse.md)。

后端依赖方向保持 `Router -> API -> Service -> Model`：

- Router 只挂载 `/warehouse` 私有路由及现有 JWT、Casbin 中间件；API 负责 HTTP 参数和统一响应。
- Service 承载事务、版本、状态、引用及快照规则；Service 接收已提取的操作者参数，不依赖 `gin.Context`。
- Model 定义持久化结构、请求 DTO 和响应 DTO；请求 DTO 不能直接复用数据库模型。
- `enter.go` 是各层的组装入口；`initialize/` 只负责插件初始化，不承载业务校验。

## 依赖

- 后端基础依赖框架的 `global.GVA_MODEL`、插件 v2 生命周期及现有 JWT/Casbin；十进制规则使用 `shopspring/decimal`，助记符建议使用 `go-pinyin`。
- 持久化部署目标是 MySQL；活跃唯一性依赖 MySQL 可空生成列与唯一索引，不能退化为仅应用层查重。
- 前端仍为规划，落地时复用项目动态菜单、请求封装及 `web/src/utils/`，不添加静态业务路由。
- 物料图片将依赖现有系统文件与 OSS 抽象，但当前未接入；送货历史通过稳定 ID 与快照关联，不依赖可复用业务编号。

## 当前基础与未落地能力

当前已落地的是基础模型、十进制/治理辅助函数、领域错误、单元测试、插件初始化、显式 MySQL 迁移描述和可复用事务边界，以及单位、品牌、类别和助记符建议的 Service、HTTP 路由与 Swagger。模型已声明 `wh_` 表名；初始化在 `AutoMigrate` 后只对 MySQL 5.7+/MariaDB 10.2+ 幂等补充可空生成列及索引。

迁移只会添加缺失的列和索引，不执行 `DROP`、物理删除或业务数据修订。其活跃唯一键将有效记录的规范化键存入生成列，将软删除记录映射为 `NULL`，再以唯一索引约束；这不是 `(code, deleted_at)` 组合索引的替代品。数据库版本不支持这些生成列时，初始化记录部署错误而不静默退化为应用层查重。

已落地的业务 HTTP 范围仅为单位、品牌、类别的 list/detail/create/update/delete、类别树和助记符建议；其余资源均未注册端点。Excel 模板、导入和导出尚未落地。

以下能力仍是计划，不能视为已实现：

- 物料、送货单、编号预占、审核、方案或报表的 Service 和 HTTP 端点；
- 菜单、API 权限、前端 API 封装、页面、组件和动态路由；
- Excel 模板、导入、导出、图片上传及真实 OSS 处理；
- 对账查询、打印投影和共享方案管理；
- 真实 MySQL 上的迁移执行、生成列/索引锁语义、行锁及隔离级别验证。单元测试只能检验 DDL 构造与内存事务回滚语义，不能证明上述部署行为。

## 目录边界

| 路径 | 边界 |
| --- | --- |
| `server/plugin/warehouse/model/` | 已有业务表结构和公共 `BusinessModel`；未来的 request/response 只表达显式 DTO。 |
| `server/plugin/warehouse/service/` | 已有十进制、审计、版本、助记符及单位/品牌/类别业务服务；其余业务命令仍未实现，且 Service 不依赖 HTTP 上下文。 |
| `server/plugin/warehouse/api/`、`router/` | 已注册单位、品牌、类别和助记符 HTTP 适配/路由；物料、送货单、报表等尚无处理函数。 |
| `server/plugin/warehouse/initialize/` | 已初始化路由和受约束迁移；菜单/API 权限在具体资源落地前不创建记录。 |
| `web/src/plugin/warehouse/` | 规划中的前端插件目录，当前没有业务页面或接口封装。 |

## 公共记录约束

所有业务持久化记录应嵌入 `BusinessModel`。它继承 `global.GVA_MODEL`，因此 `ID`、`CreatedAt`、`UpdatedAt`、`DeletedAt` 继续遵循框架约定；`DeletedAt` 是软删除的唯一事实来源，响应所需的 `isDeleted` 只能由它计算，不能另存布尔列。

`BusinessModel` 还保存服务端拥有的 `CreatedBy`、`UpdatedBy`、可空的 `ReviewedBy`/`ReviewedAt` 与非空 `Version`（默认 `1`）。客户端、Excel 和前端不得写入审计字段；基础资料、报表方案和技术编号记录不走审核，审核字段保持为空。版本在成功的修改、删除、审核或反审核命令中只递增一次；请求中的 `version` 可省略，但显式传入必须为正数且等于当前值。

记录关联一律使用稳定 ID，而不是可软删除后复用的业务编号。资料的活跃唯一性、引用保护、主从聚合软删除及并发锁定均属于后续 Service 与 MySQL 迁移共同保证的业务约束，当前模型声明不能单独证明这些行为。

## 基础资料字段、唯一性与类别树

所有资料字符串长度按 **UTF-8 字符数** 校验，不能让数据库截断：单位 `name` 最多 100；品牌和类别的 `name`、`mnemonic` 最多 100、`code` 最多 64；物料 `name` 最多 200、`specification` 最多 255、`alias` 最多 200，物料 `mnemonic` 最多 100、`code` 最多 64。

活跃记录的查重范围是：单位 `name`；品牌 `name`、`code`；类别 `code`、`(parentId,name)`；物料 `code`；物料单位绑定 `(materialId,unitId)`；个人方案 `(ownerId,name)`；共享方案 `name`。比较前去除首尾空白并按大小写不敏感比较；仅未软删除记录参与这些唯一性约束，因此软删除资料可复用编号或名称，历史关联仍只使用稳定 ID 和快照。送货单号、编号日期和预占 token 不因软删除复用。

类别的 `parentId=0` 是虚拟根，不是一个可引用的真实类别。类别编号不得使用保留值 `ROOT`；planned Excel 中父级为根时用 `ROOT` 表示，真实类别编号不能用它。类别树应一次读取有效节点后在内存组装，不能逐节点递归查询。将父类别作为筛选条件时，应以该节点及全部后代 ID 限制结果；未传父筛选时仍是未筛选查询，不能悄悄改成某个子树。

## 模型表清单

以下 `wh_` 表已有 Go 模型声明；是否具备完整部署级 DDL 仍以显式迁移完成情况为准。

| 表 | 模型 | 业务用途 |
| --- | --- | --- |
| `wh_units` | `Unit` | 公共单位资料。 |
| `wh_brands` | `Brand` | 品牌及编号、助记符。 |
| `wh_categories` | `Category` | 类别树；`parent_id=0` 表示虚拟根。 |
| `wh_materials` | `Material` | 物料及品牌、类别、可选规格/别名/图片文件引用。 |
| `wh_material_units` | `MaterialUnit` | 物料与单位绑定、换算系数、基本单位及启用状态。 |
| `wh_material_images` | `MaterialImage` | 已通过专用校验的系统文件登记，不是独立图片 CRUD。 |
| `wh_delivery_notes` | `DeliveryNote` | 送货单、不可变单号、品牌与创建/审核人快照、状态和总金额。 |
| `wh_delivery_details` | `DeliveryDetail` | 送货明细及物料、类别、单位、系数和金额快照。 |
| `wh_number_counters` | `NumberCounter` | 按业务日计数的技术记录；不提供人工修改或删除入口。 |
| `wh_number_reservations` | `NumberReservation` | 正式单号预占及消费关联；不提供回收或删除入口。 |
| `wh_report_schemes` | `ReportScheme` | 个人或共享的有序列方案。 |

## 数值、时间和快照

金额及数量相关 JSON 字段必须使用规范十进制**字符串**传输；前端不得以 JavaScript `Number` 计算账务结果。模型使用 `decimal.Decimal`，数据库目标为 `DECIMAL`。计划中的固定输出精度为：数量、单价 4 位；换算系数 6 位；基本数量 10 位；金额和总额 2 位。拒绝指数表示、NaN、Infinity、超精度、越界，以及不允许的负值；单价可为 `0`，数量必须大于 `0`。

持久化时刻统一转 UTC，业务日/月按 `Asia/Shanghai` 解释；不得依赖服务器本地时区。送货单和明细保存资料快照，历史读取、审核和对账不能用当前资料名称或已变更系数重写历史快照。

## 实施阶段

1. **基础阶段（已落地）**：插件骨架、模型、领域错误、十进制规则、版本/审计和助记符基础函数及其单元测试。
2. **资料阶段（部分已落地）**：单位、品牌、类别、部门、助记符、物料聚合及基础资料 Excel 已有 Service、HTTP 接口和 Swagger；送货单 Excel 尚未接入。导入顺序是单位→品牌→类别→物料。物料单位按完整集合更新并保留匹配 `unitId` 的绑定 ID；有送货明细引用时不能删除绑定，但允许重新指定唯一启用基本单位，历史单位和换算快照不随之改写。物料图片仅接受 jpg/jpeg/png/bmp，最大 1MiB，不做自动压缩；物料只能引用专用上传入口登记成功的 `fileId`。
3. **数据治理阶段（已接入）**：物料/单据写入使用 GORM 事务，报表读取使用同一只读一致性视图。迁移已有幂等 MySQL DDL；本轮只读冒烟启动成功，不代表并发锁语义已验证。
4. **业务接入阶段（页面整改已交付，待用户验收）**：四类资料列表/表单/分页、送货列表与连续录入、对账筛选/列方案/汇总/导出/打印已连接真实接口。报表服务为 `reconciliation_db.go`，方案持久化为 `report_scheme_db.go`；前端共用 `api/common.js`，不再依赖 loader props 或旧的独立状态函数薄壳。

## 明确不做

本插件不重构全局权限、上传、Excel 或插件框架；不新增货币字段；不通过可复用业务编号关联历史；不提供业务物理删除、恢复、编号回收或技术编号手工维护。送货单 Excel 尚未实现；OSS、真实并发与纸张分页不在本轮必要冒烟检查范围。

## 单元测试边界

测试直接断言消费者可见的业务结果：审计与版本、软删除后的编号复用、引用保护、类别/单位约束、十进制计算、编号预占、单据状态与快照、Excel 解析、报表投影及组件交互。纯函数直接测试；数据库边界仅使用能表达回滚语义的窄内存替身。

单元测试不以 SQL 文本、函数调用次数、源码字符串或 mock 回声作为断言，也不连接真实 MySQL、Redis、OSS、浏览器或 HTTP 联调。它不能证明 MySQL 生成列和行锁、真实 OSS、生产授权或浏览器打印布局。