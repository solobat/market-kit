# 2026-11-14 北京时间巡检

主并集20,465键/20平台，Aster/Lighter/Backpack/Extended SPOT原生补充44键，总计20,509条独立逐市场处置。主输入21,858条，6条Aster空contractType另作metadata-only unresolved；每批250、最多2并发，API错误0。resolved/无flags不是单位、身份或可自动交易的独立认证。

线上30条精确修复分两批：11个Bybit、3个gTrade、1个QFEX及15个Popdex。写前完整比较、compact=1写入、两批即时回读均无差异。最终写入2026-11-14T01:09:26.127416+00:00后，成功自然同步2026-11-14T01:09:37.059974286Z→2026-11-14T01:09:38.643967962Z，1352条新旧规则回验无新增差异/覆盖丢失；相关assets显示视图回读成功。既有MTH已知空结算被旧API补USD的限制仍单列knownFieldLimitations，不冒称该字段修好、不重复写。30条新修复均达到身份/比较/结算期望；本次未改MTH。

Bybit：4STOCKUSDT/AGIUSDT/HAJIMIUSDT原生LinearPerpetual官方deliveryTime=1794560400000已过，当前原生目录已不列出，改exact ended perpetual/prohibited，保留USDT结算及明确结束时间，不因4STOCK名称改成股票。BTC/DOGE/ETH/HYPE/MNT/SOL/XAUT/XRP的13NOV26共8个原生LinearFutures明确deliveryTime=1794556800000已过，真实future查询从ambiguous改prohibited，保留既有venue/expiry身份、underlying及单位；错误perpetual类型查询此前已prohibited，未重复写。时间来自Nov13官方原始字段，不从符号后缀猜日期；旧Trading快照不是当前上市或实际结算成交的证明，只证明曾明确安排的结束时间，当前缺失作为佐证。新04DEC26的8个future仍用各自原生到期/结算、ambiguous，不与旧期限或永续合并。

gTrade：新增TTWO/USD、OPENAI/USD、ANTH/USD原生group stocks-3，但此前直接用通用股票eligible身份及USD结算。改为equity:gtrade:cfd:<raw>专属产品、equity_cfd/multi_collateral_cfd、ambiguous；平台4种collaterals，未选择抵押品，故清空结算，不把USD报价当保证金。只保留股票参考类别，不据ticker认证发行人股类/盘前单位；ANTH不自动并入io:ANTH，OPENAI不自动并其他盘前产品。QFEX新增PM-USD同时在ACTIVE/EQUITY refdata与Perpetual contracts，保留USD quote/USDC margin，改venue scoped ambiguous，未认证现金股或其他平台的单位等价。QFEX其余97条非active参考元数据仍不在当前原生合约目录，逐条metadata-only，不虚构97个已核实永续。

Popdex：当前官方Futures目录81条，其中15条新产品全部PreTrading（8 Crypto、7 Rwa），线上却全部eligible。本次15条全部专属prelaunch/prohibited；Crypto分类依原生字段，7 Rwa仅粗类别不足证明股票/发行人/单位，改unknown/rwa scoped身份，保留原始display和独立settlementToken，不从quote、杠杆或名字猜倍率。全部81条原生目录有单独review记录，最终15条用同步后结果覆盖旧snapshot。新增原生status检查--official-popdex加入每日全量审计，标记仍是核查线索，不自动建立别名或可比单位。

防复发：原Popdex适配器对任何未列入硬编码清单的Rwa直接return error，7个新品会触发整个目录失败。现在未知Rwa保留unknown及官方metadata，不丢弃其他Crypto/已知商品/ETF；生成层未知Rwa独立ambiguous、原生PreTrading/Delisted禁止比较，且优先于弱consumer live/stock提示。结算来自settlementToken；缺失或冲突保留未知并抑制quote自动回填，错误source URL不能当原生证明。已知活跃Crypto行为不变，已有商品/ETF分类测试保留。schema17使部署后拒绝旧动态生成语义。代码路径可由官方新品和回归复现；当前聚合Popdex缺失与之相符，但线上缺collector失败遥测，不能断言生产缺失只有这一原因。

主并集新增30键、缺失33键：Bybit8个新future+IDOL/OROCHI，gTrade3，QFEX MTH/PM（MTH昨日快照外修复今日进入registry，不是新上市），Popdex15。缺失33均Gate当前目录/来源不再提供，不能仅凭缺失断言下架时间或错误删除历史注册规则。Gate官方35个符号缺失，其中已注册审核边界可保留；不自动把缺失当改名。Aster GLW/LLY/STRC从PRE_SETTLE回TRADING，但现有prohibited保留等待身份/单位核查；Binance历史修复、Reality包装、ADR/普通股、US100等边界未放行。Bitget REMBJ/RROKU转limit_open，仅记录交易限制，不把它当价格单位变化；ALPINE/QUBIC现货目录缺失不猜转换关系。HIVE/COPPER等旧待核实项保留。

当日官方覆盖Extended416 perpetual+4 spot、gTrade496全部组别、GRVT184、GMX126、Bybit linear完整分页、Binance USD-M、Gate futures、Hyper全部DEX/spot、Bitget SPOT/futures、QFEX refdata+contracts、Variational、Aster、Lighter、Backpack、Arcus、Popdex。OKX SPOT/SWAP仍TLS EOF，未使用旧快照冒充今日独立核实；当前registry/discovery键resolve仍检查。聚合inventory19545行vs19595，当前源平台18个，GMX与Popdex缺失，用各自今日官方目录补查；Lighter-Robinhood71 consumer键有结果但独立provider原生目录仍未核实。源变化不能当成新上市或确切collector故障，所有来源URL/时间及失败保留fetch-manifest、supplemental-fetches、popdex-fetch。

验证：完整Go套件排除既有TestHandleDiscoveryLookupAggregatesAllDiscoverySources后通过；该用例单独仍GroupCount3 vs2，未改断言/聚合器。未知Rwa不破坏目录、原生状态胜弱source两种顺序、Crypto保持、collateral冲突不猜quote、官方source边界均有回归；vet、构建通过，Python11项通过，diff检查通过。21939行本地回放（21858主证据+81原生Popdex）：sync wall1004ms、CPU1031.9ms、读p50 1292ns/p99 4000ns、heap143MiB。与完整测试并行，非同输入生产前后对照，不宣称CPU或尾延迟改善。新增metadata/身份决策只在现有采集/生成周期，无新每tick循环、订阅、队列或全量序列化；未改变刷新、丢弃或陈旧策略，无JS事件循环改动。

线上仍4efa310/schema10，本地schema17及本日持久规则待推送部署；MTH旧版结算回填问题仍是部署待办。未推送、部署、重启或暂停同步，保留原未跟踪日志与状态。两批原始写前/请求/响应/回读独立保留，combined online-write仅最终manifest；所有逐键处置保留原snapshot时间与preRepairFlags，不能把缓存审计当写后实时验证。
