# Design

## Context

本文件是实施参考，不代表功能已经存在。动机和范围见 [proposal.md](proposal.md)；行为验收以四份 `specs/*/spec.md` 为准。

### 已观察的仓库事实

| 位置 | 可复用机制及限制 |
| --- | --- |
| `AGENT.MD:35-50` | `Router -> API -> Service -> Model`；统一响应、分页；跨栈契约更新到 `aiDoc/frontend-backend/` |
| `server/global/model.go:8-14` | `GVA_MODEL` 提供ID、创建/修改时间、软删除；缺少操作者、审核和版本 |
| `server/plugin/announcement/plugin.go:20-31` | 插件分开初始化API、菜单、数据库和路由 |
| `server/plugin/announcement/initialize/router.go:10-14` | 插件私有路由挂JWT与Casbin，不把业务写接口放在public组 |
| `server/middleware/casbin_rbac.go:13-30` | 按角色、路径、HTTP方法检查权限 |
| `server/service/system/sys_export_template.go:613-724` | Excel导入直接映射/写表，不执行本业务Service校验 |
| `web/src/view/example/customer/customer.vue` | 标准表格、分页、抽屉表单模式 |
| `web/src/view/example/upload/upload.vue` | 左树右表布局、类别筛选模式 |
| `web/src/utils/request.js`、`asyncRouter.js`、`btnAuth.js` | API、动态插件页面和按钮权限入口 |
| `web/src/components/upload/image.vue:52-65` | 当前仅JPEG/PNG，超限会压缩；不可直接用于本次严格上传 |
| `web/src/utils/doc.test.js` | 已有Node原生单元测试；`web/package.json`无业务测试脚本和Vitest依赖 |

现有业务目录未发现本次物料、送货单、对账实现。当前运行配置选择MySQL和本地文件存储；本规划不读取或复制连接密钥。未验证MySQL服务器版本及生产数据规模。

### 决策来源

| 来源 | 已确认内容 |
| --- | --- |
| 原文 | 基础字段、基础资料Excel、物料字段及图片、类别树、送货单操作、参考图交互、仅送货单对账 |
| 本轮确认 | 所选维度分组；仅已审核及历史快照；唯一基本单位且明细单位必选；审核后锁定；新建预占正式号 |
| 本轮确认 | 整单同品牌，切换确认清空；编号全局唯一，软删除后可复用；名称唯一规则与虚拟根；引用删除保护 |
| 本轮确认 | Excel支持更新且整文件回滚；编号定位，送货明细及物料单位完整集合替换；个人与共享方案 |
| 本轮确认 | 数量4位、单价4位、系数6位、金额2位；可选版本传入则校验；Enter/Tab提交和多选物料弹窗 |

### 设计约定，不冒充用户原话

- 使用业务时区 `Asia/Shanghai`，接口时间为带时区RFC3339；所有持久化时刻统一转UTC存入 `datetime(6)`，读写不依赖服务器Local设置。业务日/月按上海时区转换；SQL按明确的UTC到`+08:00`转换分组，不依赖MySQL命名时区表。编号日来自服务器业务日，不来自送货时间。
- `1MB` 按 `1,048,576` 字节处理，边界值允许。全系统暂按一种记账金额处理，不新增币种字段。
- 编号/名称唯一键去除首尾空白并按大小写不敏感比较；保留用户规范化后的原始显示值。不做全角半角或模糊去重。
- 基础资料编号由人工输入；助记符自动建议但可手填。类别和品牌同样提供首字母建议。
- 共享方案通过单独的管理权限维护；个人方案即使管理员也不能越过所有者读取。未选方案时使用内置默认列，不持久化默认方案。
- 不增加审核历史表；只保留当前审核字段及框架操作日志。反审核清空当前审核字段。
- 图片替换不立即删除旧存储对象，避免其他引用被破坏；不增加图片垃圾回收任务。

## Goals / Non-Goals

**Goals:**
- 给实施者确定的文件职责、请求/响应、字段精度、索引、Excel映射、状态流转和测试预期。
- 页面与导入共用业务校验；报表列表、导出、打印共用查询投影，避免三套统计实现。
- 以业务插件隔离新增能力，不将框架示例改造成业务模块。

**Non-Goals:**
- 不升级框架或重构全局权限/上传/Excel模块。
- 不为所有框架数据库驱动承诺兼容，只以当前MySQL作为部署目标。
- 不用浏览器、真实数据库、Redis、OSS、HTTP联调或端到端测试作本变更功能验收。
- 不因规划需要执行代码生成、业务迁移、菜单注册或授权；这些属于未来apply。

## Decisions

### D1. 业务插件与文件边界

采用 `warehouse` 插件，复用现有插件v2注册方式。相比散落主工程的多套模块，插件使初始化和业务边界集中；不再引入第二个插件框架。

拟新增文件均位于以下目录，表中路径是规划路径，不是现存文件：

| 路径 | 职责 |
| --- | --- |
| `server/plugin/warehouse/plugin.go` | 按现有插件生命周期调用初始化；不放业务校验 |
| `server/plugin/warehouse/model/` | `base.go`、`unit.go`、`brand.go`、`category.go`、`material.go`、`material_image.go`、`delivery_note.go`、`report_scheme.go`、`numbering.go` |
| `server/plugin/warehouse/model/request/` | 显式创建/更新DTO、列表筛选、Excel及报表请求；不直接绑定数据库Model |
| `server/plugin/warehouse/model/response/` | 明细、分页、列元数据、错误行及汇总结构 |
| `server/plugin/warehouse/service/` | 按资料、送货单、编号、Excel、报表、方案和图片拆文件，以 `enter.go` 组合 |
| `server/plugin/warehouse/api/`、`router/` | 参数/认证/响应与路由注册，均以 `enter.go` 组合，不在API实现事务 |
| `server/plugin/warehouse/initialize/` | `gorm.go`、`api.go`、`menu.go`、`router.go`；幂等初始化，禁止覆盖用户菜单配置 |
| `web/src/plugin/warehouse/api/` | 按 `master.js`、`delivery.js`、`report.js` 封装请求 |
| `web/src/plugin/warehouse/view/{unit,brand,category,material,delivery,reconciliation}/index.vue` | 六个业务页面，动态菜单组件路径为 `plugin/warehouse/view/.../index.vue` |
| `web/src/plugin/warehouse/components/` | `MaterialUnitEditor.vue`、`MaterialPicker.vue`、`DeliveryDetailEditor.vue`、`ReportSchemeEditor.vue`、`ReportPrintView.vue` |
| `web/src/plugin/warehouse/utils/` | 纯函数：明细状态/键盘导航、十进制字符串校验、打印投影和图片规则 |

业务接入在现有 `server/plugin/register.go` 加匿名导入；前端不加静态路由。依赖文件和测试配置按D4、D13修改。遵循已有 `gva-search-box`、`gva-table-box`、Element Plus、UnoCSS。交付时补业务模块说明及前后端契约索引。

### D2. 公共审计、状态和删除契约

所有新增持久化业务记录嵌入插件内 `BusinessModel`：

| 字段 | 数据类型与规则 |
| --- | --- |
| `id` | 沿用框架自增ID；关联使用ID，禁止以可复用编号充当外键 |
| `createdAt` / `updatedAt` | 沿用 `GVA_MODEL`；创建时两者相同 |
| `deletedAt` | 沿用 `gorm.DeletedAt`；作为软删除唯一事实来源，不再加可失配的布尔列 |
| `isDeleted` | 如响应需要则由deletedAt计算，不独立持久化 |
| `createdBy` / `updatedBy` | 当前用户ID；API提取用户上下文后作为参数传给Service，Service不依赖 `gin.Context` |
| `reviewedBy` / `reviewedAt` | 可空；基础资料、方案和技术编号记录不审核，保持空值 |
| `version` | `uint64`，数据库非空默认1；请求为可空整数指针以区分省略与0，显式0非法 |

修改、删除、审核、反审核成功时版本递增一次；同一命令不因修改多个子项使主表重复递增。子记录新增为1，实际修改/软删除时分别递增。不可变审计字段不接受客户端写入。

写入顺序：按固定主键顺序锁定相关主记录，校验可选版本、状态和引用，执行全部变更，提交。更新字段使用显式白名单而非全量 `Save`；零单价和清空可选字段不能被GORM零值忽略。并发影响相同对象的编辑/审核必须串行检查，不能“先查询、无条件更新”。

引用保护覆盖未删除资料、未删除物料单位绑定和未删除送货明细，不区分单据是否审核。删除品牌/类别/单位时锁定该记录，创建引用的路径同样锁定引用对象，避免检查后插入引用的竞态。被引用绑定只可停用；被引用物料不允许改基本单位，防止历史基本数量不可比较。允许改辅助单位系数，旧单快照保持不变。

物料与自身单位绑定构成一个聚合：没有外部有效单据引用时，删除物料应在同一事务软删除其绑定，不能把自身子记录当作外部引用使物料永远无法删除；公共单位本身不删除。送货单与明细采用相同的父子聚合软删除原则。

### D3. 业务表和字段

表名显式使用 `wh_` 前缀，避免ORM自动命名差异。下表均额外包含D2公共字段。字符串长度按UTF-8字符校验，不把过长值交给数据库截断。

#### 基础资料表

| 表 | 额外字段 |
| --- | --- |
| `wh_units` | `name varchar(100)` 必填 |
| `wh_brands` | `name varchar(100)`、`mnemonic varchar(100)`、`code varchar(64)` 必填 |
| `wh_categories` | `name varchar(100)`、`code varchar(64)`、`mnemonic varchar(100)` 必填；`parent_id` 非空默认0，0表示虚拟根 |
| `wh_materials` | `name varchar(200)`、`mnemonic varchar(100)`、`code varchar(64)`、`brand_id`、`category_id` 必填；`specification varchar(255)`、`alias varchar(200)`、`image_file_id` 可空 |
| `wh_material_units` | `material_id`、`unit_id`、`factor decimal(18,6)` 必填；`is_base bool`、`enabled bool`；新系数默认1 |
| `wh_material_images` | `file_id` 关联现有系统文件且唯一；`mime varchar(32)`、`size_bytes`、`validation_version`；只有通过专用校验后才能创建，无独立业务CRUD |

类别可移动，不能指向自身或后代。树查询用一次拉取有效节点后组装，不对每个节点递归查询数据库。父筛选使用后代ID集合；限制到已选类别子树，不更改“查询全部”的含义。类别改上级时按ID顺序锁定现有有效树节点并重新验证成环，避免并发互移导致循环；新建节点仍由父记录锁和唯一键保护。

物料单位更新按 `unit_id` 对齐保留原绑定ID，不能全部删除再重建。基本单位必须启用且系数1；删除/停用旧基本单位前，提交集合须包含新的合法基本单位且物料未被有效单据引用。辅助绑定可以启停；停用绑定的旧单若未改变物料/绑定，仅改数量/单价可保存，重新选择停用绑定则禁止。

助记符建议在后端统一提供，拟采用 `github.com/mozillazg/go-pinyin`：汉字取库的默认读音首字母，拉丁字母/数字保留并小写，空白/标点忽略；多音字允许用户手改，不承诺语义消歧。未能产生建议时保持空并要求手填。前端仅在助记符未手改时使用建议；显式“重新生成”可覆盖。

#### 送货单与编号表

| 表 | 额外字段 |
| --- | --- |
| `wh_delivery_notes` | `number varchar(14)` 不可变且全历史唯一；`delivered_at datetime`；`brand_id`；`brand_code_snapshot`、`brand_name_snapshot`；`status` 为 `draft/reviewed`；`total_amount decimal(38,2)` |
| `wh_delivery_details` | `note_id`、`line_no` 从1开始；`material_id`、`material_unit_id`、`unit_id`、`base_unit_id`、`category_id`；编号/名称/规格、类别编号/名称、单位名/基本单位名快照；`factor decimal(18,6)`；`quantity decimal(18,4)`；`unit_price decimal(18,4)`；`base_quantity decimal(38,10)`；`amount decimal(38,2)` |
| `wh_number_counters` | `business_date date` 全历史唯一；`last_value` 0..999999；不提供删除或手工修改入口 |
| `wh_number_reservations` | `number varchar(14)` 全历史唯一；`token` UUID随机串唯一；`owner_id`；`consumed_note_id` 可空；不提供删除或回收入口 |

`SK`+6位日期+6位流水总长14。编号技术记录也具备公共审计字段，但不属于可配置业务资料，不开放Excel或CRUD。送货单号与资料编号的软删除复用规则不同：单号永不复用。

#### 方案表

`wh_report_schemes`：`name varchar(100)`、`scope personal/shared`、`owner_id`、`columns_json` 有序列ID数组。`owner_id` 在共享方案中保留发布者身份，但不决定管理权限。筛选值、生成SQL、独立分组表达式均不存储。

#### 唯一键与查询索引

- 活跃唯一：品牌/物料/类别 `code`；单位/品牌 `name`；类别 `(parent_id,name)`；绑定 `(material_id,unit_id)`；个人方案 `(owner_id,name)`；共享方案 `name`。
- 实现活跃唯一键使用MySQL可空生成列：有效行生成规范化键，deleted_at非空生成NULL，随后加唯一索引。类别/绑定可组合父ID或物料ID，个人/共享方案用各自生成键。不要用 `(code,deleted_at)` 代替，NULL会使有效重复记录漏拦截。
- 同一物料的基本单位用“有效且is_base”的可空 `material_id` 唯一键保证至多一个，再由事务校验保证至少一个且启用。切换基本单位时先在事务内取消旧基本标记，再设置新标记，最终整体校验，不能因中间态撞唯一键。其他唯一名同样只约束有效记录。
- 单号、编号日期、预占token不受软删除影响，使用普通唯一索引。编号计数和预占记录禁止软删。
- 查询索引：物料 `(brand_id,category_id,deleted_at)`；类别 `(parent_id,deleted_at)`；单据 `(status,delivered_at,deleted_at)`、`(brand_id,delivered_at)`、`reviewed_at`；明细 `(note_id,deleted_at,line_no)`、`material_id`、`material_unit_id`、`category_id`、`unit_id`、`base_unit_id`。所有删除保护查询必须能命中相应引用索引。
- 生成列和唯一索引通过插件幂等迁移显式管理，不能假定 `AutoMigrate` 自动创建。上线前要求MySQL支持所用生成列语法；不支持则停止部署并报告，不能退化为仅应用层查重。

### D4. 精确数值与快照变更

拟使用 `github.com/shopspring/decimal` 完成十进制解析、乘法、舍入和求和，避免二进制浮点。数据库用DECIMAL，JSON全部数值金额以规范十进制字符串传输，前端不使用 `Number` 计算账务结果。数量和价格最大14位整数、系数最大12位整数；基础数量最大28位整数、金额及总额最大36位整数，超过字段范围即明确报错。

| 值 | 输入/输出规则 |
| --- | --- |
| 数量 | 必填，`>0`，最多4位小数，输出固定4位 |
| 单价 | 必填，`>=0`，最多4位小数，输出固定4位 |
| 换算系数 | `>0`，最多6位小数，输出固定6位 |
| 基本数量 | 数量×系数，精确保留10位，不中途按4位舍入 |
| 行金额 | 数量×单价，十进制half-up到2位 |
| 单据/报表金额 | 先取得每行已舍入金额，再求和，输出固定2位 |

拒绝指数表示、NaN、Infinity、超精度、越界和负号（价格可为0）。表单及Excel不默默补缺失价格为0。完全空白尾行不提交，部分填写行必须报错，不能偷偷丢弃。

首次创建快照从有效资料读取；主表存品牌快照，明细存物料、类别、单位、基本单位及系数。主表明确增加 `created_by_name_snapshot varchar(100)`、`reviewed_by_name_snapshot varchar(100)`：创建人/审核人显示名在对应业务动作时记录，避免用户改名影响历史显示。反审核同时清空当前审核者名称快照。

未审核单更新规则：
1. 以明细ID匹配旧行，确认属于该单；同物料、同绑定且表头品牌未变时保留旧快照，仅重算数量/价格结果。资料后来调整品牌或类别，不使该旧行在只改数值时失效；新增/换物料行则按当前品牌严格校验。
2. 物料或绑定改变时按当前资料重新生成该行快照；物料未变仅换辅助单位时保留物料/类别身份快照，更新单位/系数快照。
3. 新增行获取当前快照，缺失旧行软删除。非空未知明细ID报错，禁止把别单明细移入。
4. 品牌改变时页面确认清空；后端仍逐行验证新品牌，主表品牌快照仅在品牌身份改变时更新。
5. 审核检查保存快照完整、数值正确及引用未删除，不因为资料改名或旧绑定已停用而刷新/拒绝旧快照。反审核后继续保留旧业务快照。

### D5. 正式编号预占和主从状态

选择数据库日计数器，而非 `MAX(number)+1` 或Redis锁，保证多实例时编号约束只有一个事实来源。

1. 新建页确认具有新建权限后调用 `POST /warehouse/delivery/reserveNumber`。
2. 短事务创建或锁定当日计数器行，增加流水，检查不超过999999，生成正式号及随机token，保存owner和预占记录，提交后返回。
3. 保存新单传 `reservationToken`。锁定预占记录，验证所有者及未消费，校验业务后写入主从并设置 `consumed_note_id`，在同一事务提交。
4. 提交失败时预占仍可由同一用户重试；已消费token再次提交返回业务冲突，不重复创建。编辑已有单不申请新编号。
5. 关闭新建页不回收。跨日保留原号。日流水耗尽时拒绝新预占，不增加位数、不换前缀。

并发首次创建日计数器使用唯一日期键加“冲突不更新”的插入，再 `FOR UPDATE` 读取；不通过重试业务命令掩盖事务失败。新单导入按分组预先申请内部预占，校验业务及消费预占放到整文件写事务；失败允许保留未消费预占，但不能保留业务主从。

状态转换只有以下四种合法动作：

| 动作 | 前置状态 | 成功结果 |
| --- | --- | --- |
| 新建 | 无 | draft、审核字段空、version=1 |
| 编辑/删除 | draft | 保持draft或主从软删，version+1 |
| 审核 | draft | reviewed，写审核人/时间，version+1 |
| 反审核 | reviewed | draft，清空审核字段，version+1 |

重复审核/反审核返回 `STATE_CONFLICT`。审核/反审核不允许顺带修改明细。状态、可选版本、数据完整性检查在锁定单据后完成，避免编辑与审核互相穿透。

### D6. HTTP契约和错误

插件前缀 `/warehouse` 还需接在框架 `RouterPrefix` 后。API声明以最终实际路径为准，权限记录的Method与路径必须一致。DTO只接受明确字段，ID为正整数，版本可省略。

通用成功JSON：`{code:0,data:...,msg:"操作成功"}`。列表 `data` 包含 `{page,pageSize,total,list}`，默认page=1、pageSize=20，允许1..100；报表额外返回列、全局合计和查询说明。无记录详情返回业务错误，不构造空对象成功响应。

| 资源/方法 | 路径与请求 |
| --- | --- |
| 资料列表/详情 | `GET /master/{units,brands,categories,materials}/list`、`/detail?id=`；类别额外 `/tree` |
| 资料新增/更新/删除 | `POST .../create`、`PUT .../update`、`DELETE .../delete`；删除body为 `{id,version?}` |
| 助记符建议 | `POST /master/mnemonic`，`{name}` 返回 `{mnemonic}`；只有对应资料新建/编辑权限者使用 |
| 资料Excel | `GET .../template`；`POST .../import` multipart `file`；`GET .../export` 接收同列表筛选 |
| 物料图片 | `POST /master/materials/uploadImage` multipart `file`，返回 `{fileId,url,size,mime}` |
| 送货单 | `GET /delivery/list`、`/detail?id=`；`POST /delivery/create`；`PUT /delivery/update`；`DELETE /delivery/delete` |
| 送货单动作 | `POST /delivery/reserveNumber`、`/review`、`/unreview`；后两者body `{id,version?}` |
| 送货单Excel | `GET /delivery/template`；`POST /delivery/import`；`GET /delivery/export` |
| 报表 | `GET /reconciliation/columns`；`POST /reconciliation/query`、`/export`、`/printData` |
| 方案 | `GET /schemes/list`、`/detail?id=`；`POST /schemes/create`；`PUT /schemes/update`；`DELETE /schemes/delete` |

表中 `...` 仅表示同一资料资源的完整路径拼接，实现和API注册必须展开，不创建字面通配CRUD接口。图片、助记符、引用候选列表复用当前模块新建/编辑所需最小权限，不能要求录单员获得资料维护权限才能读取下拉候选。为此提供 `GET /delivery/materialOptions` 和 `/delivery/brandOptions` 供有送货单查看/录入权限者使用；响应只给必要身份、快照候选和启用单位。报表筛选候选由 `GET /reconciliation/options` 从已审核快照去重生成，保留历史名称，不依赖当前资料名称。

创建物料DTO：`{name,mnemonic,code,brandId,categoryId,specification?,alias?,imageFileId?,units:[{unitId,factor,isBase,enabled}]}`。更新额外 `{id,version?}`，单位按完整集合处理。

创建单据DTO：`{reservationToken,deliveredAt,brandId,details:[{materialId,materialUnitId,quantity,unitPrice}]}`；更新改为 `{id,version?,deliveredAt,brandId,details:[{id?,materialId,materialUnitId,quantity,unitPrice}]}`。不接收number、lineNo、名称快照、基本数量、金额、状态或审核字段作为可信写入值。lineNo按数组顺序生成。

业务失败保持框架 `code != 0`，`data` 统一补充 `{errorCode,field?,lineNo?,currentVersion?,errors?}`。错误枚举：`VALIDATION_FAILED`、`DUPLICATE_VALUE`、`VERSION_CONFLICT`、`STATE_CONFLICT`、`REFERENCE_IN_USE`、`NOT_FOUND`、`FORBIDDEN`、`IMPORT_FAILED`、`NUMBER_EXHAUSTED`。错误消息中文；数据库原始错误和SQL不直接返回。权限认证沿用框架现有响应，不为业务另改HTTP全局拦截器。

导出成功返回带正确MIME及 `Content-Disposition` 的xlsx流，前端使用请求封装的blob能力下载；失败返回标准JSON并由下载工具识别，不能把错误JSON保存成xlsx。打印不是文件下载，返回标准JSON报表投影。

### D7. Excel模板与事务

复用 `excelize` 读写，不复用通用模板的直接写库。每类资源提供自描述模板：必填列加 `*`；编号/日期/小数列用文本格式，避免前导零、大数和时区被Excel自动改写。不导入公式单元格，导出文本以明确字符串类型写入，防止 `= + - @` 开头名称被解释为公式。

| 模板 | 工作表与列 |
| --- | --- |
| 单位 | `单位`：单位名称*、版本号（可选）；同名匹配更新，不通过导入重命名单位 |
| 品牌 | `品牌`：编号*、品牌名称*、助记符*、版本号；编号匹配更新 |
| 类别 | `类别`：编号*、类别名称*、助记符*、上级类别编号*、版本号；上级用保留值`ROOT`表示无上级，禁止真实类别编号使用ROOT |
| 物料 | `物料`：编号*、物料名称*、助记符*、规格、别名、品牌编号*、类别编号*、图片文件ID、版本号 |
| 物料单位 | 同物料文件的 `物料单位`：物料编号*、单位名称*、换算系数*（模板示例1）、是否基本单位*、是否启用*；布尔用“是/否” |
| 送货单 | `送货单`：导入分组键、单号、送货时间*、品牌编号*、明细ID、序号*、物料编号*、单位名称*、数量*、单价*、版本号；导出增加只读名称/规格/系数/基本数量/金额/审核及审计列 |

类别的同文件新父类别允许前向引用：先解析全量类别编号，构图检测未知父/重复编号/循环，再按拓扑顺序写入。其他资源不跨文件自动创建依赖，按单位→品牌→类别→物料→送货单的顺序导入。

送货单分组规则：有单号按单号分组；无单号要求分组键，按该键分组，新建组内禁止混入非空单号。同一分组送货时间、品牌及版本重复填写且必须完全一致，包括版本省略状态。序号要求组内为1..N无重复；同一物料允许多行，不能按物料编号去重。

导出保留明细ID以匹配旧快照。更新文件：非空明细ID必须属于该单且与同一行物料/绑定一起校验，空ID表示新行；未出现旧ID表示软删旧行。没有旧ID的纯手工更新文件按全部新行处理并取当前快照，模板说明其与保留ID更新的差异，不能猜测按序号匹配历史行。

资料单位集合：物料主表每个编号只出现一次，单位工作表必须恰好覆盖文件中各物料，出现未知物料编号、重复绑定或无基本单位即失败。空白可选列表示清空，不是“保留旧值”；主键匹配列及必填列不得为空。明确区分空白版本与版本0。

导入阶段：
1. 解析文件、表头、文本、小数、组内一致性，收集 `{sheet,row,column,reason}`。
2. 批量读取所需有效资料及现存目标，完成基础引用检查；无效直接返回，不写业务数据。
3. 新送货单组需要正式号时先预占，不占有长事务期间的计数器锁。
4. 单个文件一个业务事务，固定顺序锁定更新目标及引用对象，重新验证版本/状态/唯一性，调用不自行开启第二事务的内部业务保存函数。
5. 任意失败回滚全部业务修改；响应错误列表。成功返回 `{createdCount,updatedCount,numberMappings}`，单据计数按组而非明细行。

导出按当前筛选导出全部有效记录，不受页面分页影响。物料导出包含完整单位工作表；送货单导出基于历史快照。更新型回导可能因旧关联已失效而合法失败，不能为“可回导”绕过引用或审核限制。

### D8. 动态列和统一报表投影

采用固定列注册表，前端从 `/reconciliation/columns` 获取 `{id,label,kind,dataType,scale}`，不复制后端统计枚举。后端每列显式绑定受控表达式，不把客户端列名直接拼SQL。

| 列ID | 类型/来源 |
| --- | --- |
| `deliveryDay` / `deliveryMonth` | 维度；送货时间在业务时区的日/月，不额外提供年度配置 |
| `noteNumber` / `brand` | 维度；单号、品牌ID与编号/名称快照 |
| `materialCode` / `materialName` / `specification` | 维度；物料ID及对应快照值；只选规格不隐含物料分组 |
| `category` / `unit` | 维度；类别ID及名称/编号快照、单位ID及名称快照 |
| `creator` / `createdAt` | 维度；单据创建者身份/名称快照、单据创建时间 |
| `reviewer` / `reviewedAt` | 维度；单据当前审核者身份/名称快照、审核时间 |
| `quantity` / `baseQuantity` | 指标；分别按物料+所选单位、物料+基本单位可比检查 |
| `unitPrice` | 条件值；组内物料、所选单位和单价全部一致才输出 |
| `amount` | 指标；求和已按行舍入的金额 |

只有materialCode/materialName列选择时附带物料身份；specification只是文本维度，否则只选规格会悄悄按物料分组。品牌/类别/单位/人员名称维度包含对应ID及快照显示值，避免同名不同身份混淆。显示相同但身份不同的维度值在表格提示编号或ID以便区分。

查询请求：`{filters:{deliveredFrom?,deliveredTo?,reviewedFrom?,reviewedTo?,number?,brandId?,categoryId?,materialKeyword?},columns:[...],page:1,pageSize:20}`。时间范围为 `[from,to)`；日期选择器把结束日期转换为次日00:00，不使用23:59:59丢失小数秒。类别筛选按快照 `category_id` 精确匹配，不用当前类别树改变历史归属；与物料维护页的“含后代”筛选明确不同。物料关键字匹配快照编号或名称。

内置默认列：送货日、单号、品牌、物料编号、物料名称、规格、单位、数量、单价、金额。加载方案只改变有序列集合，不改变筛选值。查询发出时捕获列快照，响应对应此快照；不能以请求期间正在变化的页面列解释旧响应。

投影步骤：
1. 校验列非空、无重复且均在白名单；筛选约束基础已审核未删除数据集。
2. 已选dimension列共同决定分组键。相同资料身份但快照值不同，按显示值拆组，不将历史名称覆盖为新名称。
3. 使用精确DECIMAL聚合amount。quantity只有组内material_id与unit_id各自唯一时求和；baseQuantity要求material_id与base_unit_id各自唯一。
4. unitPrice要求material_id、unit_id及价格都唯一，其他情况值为null并附原因。隐藏quantity不妨碍按需计算unitPrice的可比条件，但不得对外输出隐藏指标。
5. 分组完成后做分页；`total` 是分组数。所有全局金额、数量桶从完整筛选数据集计算，不从当前页累加。

建议用受控SQL表达式聚合，金额比较不经过float。数量/单价可比检查通过分组内稳定ID的最小值和最大值相等及价格一致性表达，不依赖前端猜测。报表QueryBuilder专门负责白名单映射、参数绑定、分组与排序，没有通用SQL DSL。

默认排序按所选维度列顺序、规范化值及身份作稳定排序；单号字典序等同日流水序。有序维度都相同时已经属于同一组，不再以隐藏明细ID切组。只选指标时有数据返回一个组，无数据返回空列表和零合计。

响应：`{list,page,pageSize,total,columns,summary:{amount?,quantityBuckets?,baseQuantityBuckets?},warnings}`。每行含 `values`（列ID→字符串或null）和 `unavailable`（列ID→原因）。数量桶包含 `{materialId,materialCode,materialName,unitId,unitName,value}`，不同名称快照以清晰标签区分，但不能把所有桶再加成一个总数量。baseQuantity列标题为“基本数量”，值附基本单位显示元数据。

同一 `BuildProjection` 逻辑由query/export/printData共用；导出和打印禁用分页但保留同样列、分组、顺序和汇总。数据库读取在一次请求中采用同一只读事务的一致性视图，避免查询行与合计跨审核状态。不同请求仍为实时结果，不承诺用户先查后导出期间数据不变。

#### 聚合验收样例

输入均为品牌A、物料M基本单位“件”，同一送货月且已审核：

| 行 | 单位与系数 | 数量 | 单价 | 行金额 | 基本数量 |
| --- | --- | --- | --- | --- | --- |
| 1 | 箱，12 | 2.0000 | 30.0000 | 60.00 | 24.0000000000 |
| 2 | 件，1 | 3.0000 | 2.0000 | 6.00 | 3.0000000000 |
| 3 | 件，1 | 1.0000 | 2.5000 | 2.50 | 1.0000000000 |

- 选 `brand,amount`：一行品牌A，金额68.50。
- 选 `materialName,quantity,baseQuantity,unitPrice,amount`：一行M，数量null（混合单位），基本数量28.0000000000，单价null，金额68.50。
- 加选 `unit`：箱组数量2.0000、单价30.0000、金额60.00；件组数量4.0000、单价null、金额8.50。
- 再加入另一物料N即使单位也叫“件”，合并品牌时仍不得将M和N数量相加。

### D9. 报表方案权限和菜单

方案DTO为 `{name,scope,columns}`，更新/删除额外 `{id,version?}`。查询列表只返回自己的personal方案和所有shared方案。个人改共享、共享改个人都需要共享管理权限；改为个人时仅允许共享发布者成为自己的个人方案，不将他人共享方案强行归属自己，其他用户可“另存为个人”。

| 方案动作 | 个人方案 | 共享方案 |
| --- | --- | --- |
| 列表/详情/使用 | 仅所有者且有报表查询权限 | 有报表查询权限 |
| 创建/另存为 | 当前用户个人 | 额外共享管理权限 |
| 编辑/删除 | 仅所有者且有个人方案维护权限 | 共享管理权限，不要求为发布者 |
| 使用时临时改列 | 不写原方案 | 不写原方案 |

范围及ownership在Service再次检查，不只靠API权限。删除方案只软删方案；已打开的页面可继续按本地列集合查询，保存旧ID返回NOT_FOUND。

菜单三级分组按框架配置：基础资料→物料单位/品牌/物料类别/物料，仓库管理→送货单，报表管理→对账表。父组不指向虚构组件，按框架现有父菜单方式处理。API元数据由MCP创建后按现有机制注册；菜单有稳定name/path/component，重复初始化只补缺失项，不覆盖用户排序、标题和角色授权。

按钮标识前缀 `warehouse:<resource>:<action>`，资源为unit/brand/category/material/delivery/reconciliation/scheme；审核、反审核、导入、导出、打印、共享管理独立标识。后端JWT+Casbin为强制边界；报表query和printData分配不同权限，个人/共享方案在Service追加对象级检查。新功能不默认赋给所有角色，通过MCP `assign_api_to_role` 仅追加明确目标角色；具体业务角色由系统管理员配置，本规划不猜用户角色ID。

### D10. 页面和键盘状态

基础资料页面沿用搜索区、操作栏、表格/分页及新增编辑抽屉。类别是树表，物料左树右表，物料抽屉含单位子表，显示“设为基本/停用/启用/删除”；基本单位系数只读1。图片控件不复用压缩上传入口，使用严格校验上传接口及现有图片预览工具。

送货单列表显示编号、送货时间、品牌、状态、金额及审计；按编号/时间/品牌/状态查询。新建/编辑使用可容纳宽明细的框架全屏抽屉或页内编辑区，不用小弹窗挤压明细。预占完成前禁用保存，清楚显示载入或失败，不展示假正式号。审核单编辑器只读，反审核为独立确认动作。

明细编辑状态以稳定前端 `rowKey` 标识行，不能用数组下标持久追踪焦点。业务物料选择下拉含类别导航、编号/名称/助记符检索，弹窗含类别树、候选列表、已选列表，符合参考图但去掉仓库/库存等无关项。

| 当前字段/事件 | 下一动作 |
| --- | --- |
| 确认物料 | 填只读名称/规格、默认基本单位，焦点到单位选择；不自动跳过单位确认 |
| 确认单位 | 焦点到数量 |
| 数量Enter或Tab | 校验成功到单价，失败留在数量 |
| 单价Enter或Tab | 校验成功到下一行物料；无下一空行时只创建一次 |
| Shift+Tab | 反向到前一可编辑字段，不新增行 |
| IME composing、下拉未确认、弹窗按键 | 交给当前控件，不触发行提交 |
| blur或普通字符输入 | 更新值但不自动抢焦点 |

以用户选择提交动作而非watch每个值判断完成。事件处理只在实际移动焦点时拦截浏览器默认行为，组件更新后 `nextTick` 对定位ref聚焦，不靠document全局选择器。批量多选添加顺序稳定，同弹窗内同物料只选一次；已有单据允许同物料再次追加为独立行。数量/价格先留空以要求明确录入，不将0作为完成状态。

报表左侧方案/显示列区域可收起，右侧查询和报表。只提供已确认的列勾选/顺序，不出现冻结、独立分组、高级筛选标签。不可比数值显示“—”及原因，不把null变为0。

打印调用printData获取全量投影，在 `ReportPrintView` 中使用Vue文本插值渲染，复用列顺序和汇总组件。用打印CSS隐藏非报表区域，长表重复表头；调用 `window.print` 是功能实现，不要求本轮或后续验收连接浏览器。单元测试覆盖内容投影、转义及打印状态切换，不能据此声称真实纸张布局已验证。

### D11. 图片校验和存储

前端限制单文件、允许扩展名和1MiB，大小边界按字节比较，不自动压缩。后端限制读取大小到上限+1，校验扩展名与实际签名/解码类型一致，接受JPEG、PNG和BMP；BMP解码可使用现有 `golang.org/x/image` 能力，不改变框架公共上传行为。验证通过后再交给现有OSS抽象，不直接使用客户端文件名拼路径。

将合法图片登记到现有文件记录并返回fileId，同时创建 `wh_material_images` 校验登记。物料保存只接受在该登记表内且对应系统文件仍有效的fileId，不得仅凭系统文件ID存在就接受其他入口上传的超限或伪装文件。上传失败不替换当前物料图片；若登记失败则清理本次新上传对象，不清理用户旧图。单元测试使用字节流与内存存储替身，不访问真实OSS。

### D12. 实施MCP/Skill使用约定

本次已调用GVA `requirement_analyzer`，返回的是分析提示，不是业务已生成的证据。后续apply开始后：
1. 读本规划及仓库规则，调用 `requirement_analyzer`，按已确认规则组织模块，不接受工具建议扩展业务。
2. 使用 `gva_analyze` 核对现有包/模块。注意它可能清理空包，属于实施阶段而非当前只读调研。
3. 按依赖分批调用 `gva_execute` 生成插件基础结构，先确认能表达主从、字段精度和审计；生成器不能表达的部分按现有模式手工实现，不把全部复杂逻辑交给生成器猜测。
4. 每批生成后调用 `gva_review`，移除不符合本规格的物理删除、无条件Save和直接导入SQL。
5. 通过 `list_all_apis` / `list_all_menus` 先核对，再调用 `create_api` / `create_menu` 补齐实际资源；避免双重初始化或重复菜单。角色授权仅在已确定目标角色时追加。

当前本地 `.codex/skills/`、`.claude/skills/` 为空，不虚构框架Skill。可使用已全局安装的OpenSpec工作流Skill；如实施工具需要额外模板，以MCP实际返回为准。

### D13. 仅单元测试的验证计划

Go采用标准 `testing` 及已有断言库。抽出纯业务函数（规范化、数值、快照更新、状态判断、Excel解析、列投影）直接测真实逻辑。数据库边界只提供窄的事务/查询依赖，使用实现回滚语义的内存替身测试业务命令；不建立一套通用Repository框架，不断言SQL文本或函数调用次数。

前端保留已有Node测试，为插件新增Vitest、`@vue/test-utils`、happy-dom和单独配置，用模拟DOM做Vue组件单元测试，不启动Chromium/Playwright或CDP。Vitest版本选择与当前Vite/Vue支持范围匹配的稳定版；不为测试升级业务框架。

| 测试组 | 必须断言的消费者可见结果 |
| --- | --- |
| 审计/版本 | 创建字段不可伪造；省略版本成功递增；旧版本整次无变化；审核与编辑互斥 |
| 资料唯一与引用 | 活跃重复拒绝；软删编号复用不改变旧ID；有子节点/有效引用不能删除 |
| 类别 | ROOT、父级前向引用、自身/多节点循环、移动后树筛选 |
| 单位 | 无/多基本单位拒绝；系数精度、正数；停用保留旧快照；被引用不能删除或换基准 |
| 数值 | 0单价允许；0数量拒绝；超精度/范围拒绝；逐行舍入0.02+0.02=0.04；基本数量精确到10位 |
| 编号 | 固定时钟跨日；并发内存分配唯一；取消不回收；999999耗尽；token所有者及重复消费拒绝 |
| 单据 | 混品牌拒绝；至少一完整行；部分空行报错；快照不被改名/系数变化重写；主从错误不部分提交 |
| 审核 | draft→reviewed→draft；审核字段更新/清空；已审核任意修改入口拒绝；重复动作冲突 |
| Excel | 真实内存xlsx字节解析；表头/行号；混合新增更新；组内不一致；未知外键；旧版本/已审核整批失败；完整单位/明细替换；公式拒绝及文本导出 |
| 报表 | D8样例；增删维度改变行分组；跨物料/单位不合计；单价混合null；空集；全局汇总不取当前页；列白名单 |
| 方案 | 个人越权、共享可读不可写、发布权限、名称作用域、版本冲突、删除不影响单据 |
| 组件 | 名称建议不覆盖手填；品牌切换确认/取消；键盘Enter/Tab/Shift+Tab、IME、空尾行去重；多选取消；非法输入留焦 |
| 上传/打印/下载 | BMP/JPEG/PNG边界字节、伪装类型；超限不压缩；错误JSON不当xlsx下载；打印投影与列表列一致、文本转义 |

计划命令（仅在对应实施和测试文件存在后执行，本轮不运行）：

```bash
# cwd: server
# 包含业务子包中的全部单元测试，不连接外部服务
go test ./plugin/warehouse/...
go test -race ./plugin/warehouse/...

# cwd: web
# 原有测试保留使用原生命令
node --test src/utils/doc.test.js
# 新增脚本只运行插件单元测试
npm run test:warehouse -- --run
```

`test:warehouse` 计划指向插件专用Vitest配置，测试include限定 `src/plugin/warehouse/**/*.test.{js,ts}`，不用源码快照或查找字符串代替行为断言。不对MCP返回的注册文本、常量复制、简单转发或mock回声编写测试。

**验证限制**：单元测试不能证明真实MySQL生成列DDL、行锁隔离、SQL分组/事务在MySQL上的执行语义、真实OSS上传、生产菜单授权配置、浏览器焦点/打印纸张布局正确。内存聚合样例验证规则和结果映射，不替代数据库执行证明。实施报告必须如实列出这些未验证边界；不以“为了验证”为由违反用户不连接浏览器的约束。

## Risks / Trade-offs

- **版本可省略会允许覆盖旧值** → 标准页面始终传version，Excel模板明确空版本的含义；不擅自改为强制版本。
- **预占编号会断号、导入回滚仍可能消耗编号** → 已获用户确认，不回收、不用空单据填号。
- **软删编号复用会造成按编号关联历史的歧义** → 所有关联使用ID，历史使用快照，活跃唯一键由数据库保证。
- **只有单元测试无法验证数据库及浏览器真实环境** → 报告覆盖边界，不声称完成联调或浏览器验证。
- **显示维度越少，数量越易不可比** → 返回null和明确原因，按可比桶列出合计，不伪造一个总数量。
- **实时导出与之前列表可能不同** → 每次请求内部一致，查询后发生审核/反审核会影响后续输出，不引入未要求的冻结快照。
- **MySQL版本或报表规模尚未实测** → 迁移显式校验所需能力；先索引和实时查询，不未经测量添加记录表/缓存/异步任务。
- **单位导入完整替换可误省略旧绑定** → 模板明确语义，引用保护使危险省略失败，不直接全删重建。

## Migration Plan

以下是未来实施/部署步骤，不在本次规划执行。

1. 先完成插件领域逻辑及单元测试，再按依赖迁移审计、资料、单位绑定、编号、单据、方案和图片登记表。
2. 幂等添加生成列/唯一键和普通索引。发现活跃重复值、MySQL不支持语法或字段范围冲突时停止，不能悄悄删除已有数据。
3. 插件注册与路由启用后，创建缺失API和菜单，再由管理员分配角色；保留原有用户配置。
4. 首次业务资料按单位→品牌→类别→物料导入，送货单最后导入。不要编造品牌/物料种子数据，也不将参考图数据当真实业务数据。
5. 回滚代码时先撤销本业务入口和授权，不删除已有业务表、快照或编号；保留可恢复数据。删除业务数据/表需要另获明确授权。

## Open Questions

没有阻塞本规划的未决业务问题。生产物料数量、月单据量和MySQL实际版本尚未提供，当前不据此增加功能；实施前可据部署环境检查迁移能力，性能优化仅在有测量结果后另行决定。
