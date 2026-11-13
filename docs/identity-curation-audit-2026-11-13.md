# 2026-11-13 北京时间巡检

主并集20,468键/20平台，Aster/Lighter/Backpack/Extended SPOT官方原生补充44键，另加快照外观测的QFEX MTH查询1键，总计20,513条逐键处置。主输入21,940证据行；6条Aster空contractType另作metadata-only unresolved。每批250、最多2并发，resolve API错误0。resolved/无flags不是独立身份、单位或交易可比性认证。

线上4条精确修复：Aster GLWUSDT/LLYUSDT官方转PRE_SETTLE且原生subtype STOCK，原eligible（GLW还是泛化crypto）改股票参考产品专属pre_settle身份/prohibited；官方USDT margin保留，不全局改GLW/LLY类别或猜永续哨兵expiry。Bitget新增FRHCUSDT明确isRwa=YES/perpetual/单一USDT margin，但不足以认证发行人、股类或单位，原通用FRHC股票eligible改rwa:bitget:perpetual:frhcusdt、unknown/ambiguous，不与Binance/Gate现金股等自动合并。既有生成规则已支持这种RWA保守隔离。

QFEX MTH-USD官方refdata为INACTIVE/EQUITY，当前md/contracts全部237条Perpetual中不存在MTH；不能把预留参考条目当真实上市合约。线上快照外该具体查询仍返回显式股票eligible，已改专属unverified product/prohibited，不以参考表margin_asset或quote认证真实结算和单位。本地审核规则使用rwa_perpetual_unverified并保持空settlement。旧线上4efa310/schema10忽略该类型的空结算保护，仍将空值补成quote USD；即时写后已出现这一字段不匹配，原始online-write.errors保留，不冒称全部字段写成功，不反复重写。最终回验明确knownFieldLimitations，只针对这一字段记录实际USD，不代表认证USD保证金；已有本地防回填代码待部署。比较prohibited和独立身份已生效，MTH不具备自动比较资格。

最后写入2026-11-13T01:03:05.237226+00:00后，成功自然同步2026-11-13T01:03:37.058690427Z→2026-11-13T01:03:39.195297889Z；1330条新旧规则回验，身份/比较边界无覆盖丢失；除上述显式已知MTH字段限制外无额外差异，4个相关assets视图回读成功。MTH字段未达到清空目标，不能归为完全修复。修复前、请求、响应、即时差异及最终回读均保留；逐市场最终处置覆盖4条旧快照结果并单列MTH快照外查询。资产显示不替代具体产品边界。

QFEX全refdata334条：237 active原生Perpetual与md/contracts匹配；另97条INACTIVE/DELISTED参考条目全部不在当前原生合约目录，逐条记metadata-only，不伪装成97个已核实永续、更不批量创建虚假市场。当前缺失不证明具体历史下架日期。MTH保守禁用的是已观测消费者查询，并不认证它曾上市。STRCUSDT此次PRE_SETTLE但已有prohibited，未重复写。Binance FRHC/NU从PENDING_TRADING转TRADING，保留昨日隔离等待股类/单位核实，不能凭恢复状态自动启用。

主并集新增149键、消失0：Gate147（其中5个native股票期货/永续入口DUOL/FISV/FRHC/KMB/MS，142个杠杆现货键重新进入来源清单）、Bitget FRHC及Hyper io:NVDL。Gate5股类原生产品已venue scoped ambiguous，不误合并；杠杆现货unknown/ambiguous保留，不把名字3L/5S当底层单位倍率。Hyper io:NVDL已有prohibited，不自动与NVDA或普通股合并。Bitget RSIVEFUSDT现货目录缺失只记录观察，不假定转换比例/下架时间。来源回归不等于新上市，所有新增键有处置。

官方当前目录核对Extended416 perpetual+4 spot、gTrade493所有类别、GRVT184、GMX126；另有Binance USD-M、Bybit linear完整分页、Gate futures、Hyper全部DEX及spot、Bitget SPOT/futures、Aster、Lighter、Backpack、QFEX refdata+contracts、Variational、Arcus。原生字段大多稳定，别名/倍率、到期、包装和同名股票crypto边界保留。既有Bybit3条LinearPerpetual未来deliveryTime未变化，当前仍早于该明确时间；不提前认作已到期交割。US100、SKHY ADR/普通股、ICE/COPPER/HIVE等旧待核实边界不放行。

覆盖缺口：OKX SPOT与SWAP两次正常官方请求均TLS EOF，未下载到当日目录，不能用Nov12快照冒充Nov13官方身份核实；现有OKX registry/discovery键的线上resolve仍检查。聚合inventory19595行vs20147，当前仅15个源平台，GMX/Lighter/Lighter-Robinhood/Ondoperps/Variational均缺失；注册表保留相关键，今日官方GMX/Lighter/Variational可独立补查，Lighter-Robinhood和Ondoperps仍只有既有registry/consumer身份回验，无法宣称原生目录完整。Aster仅88源行、Backpack191，来源组成变化有逐平台delta；缺少已部署collector失败遥测，不能精确归因或称全源健康。source timestamps/errors分别保留fetch-manifest、supplemental-fetches和okx-retry-fetch。

验证identity/curation/bootstrap全部通过、嵌入数据服务构建与git diff --check通过。仅更新4条持久规则及证据，无新高频运行时逻辑，未重复CPU/事件循环测试或宣称性能收益；刷新、丢弃、队列、陈旧策略不变。既有server聚合GroupCount3 vs2历史失败见Nov12记录，本日未改其行为或测试。线上仍4efa310/schema10，本地c480e24之后的schema16及本日数据待推送部署；MTH空结算回填缺口是明确部署待办。未推送、部署、重启或暂停同步，保留原未跟踪日志和状态。
