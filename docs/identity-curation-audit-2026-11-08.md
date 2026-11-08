# 2026-11-08 北京时间巡检

主并集20,407个市场键/20个平台，Aster/Lighter/Backpack官方原生目录补充40键，Binance真实future补充4键，总计20,451条逐市场disposition；另存6条Aster缺失contractType元数据，未计为解析成功。主输入21,882条证据；专门回放加入4条官方future，21,886条。resolve每批250、最多2并发。resolved/无flags不是独立核实交易可比性。

本次新确认的系统问题：Binance USD-M 4个既有季度合约BTC/ETH×261225/270326，官方CURRENT_QUARTER/NEXT_QUARTER、baseAsset BTC/ETH、quoteAsset/marginAsset USDT、deliveryDate明确；原注册perpetual却为crypto:ticker BTC/ETH eligible，可能错误加入永续比较；future查询则落入heuristic，把日期当quote/settlement。此前巡检未覆盖这一项官方季度类型校验，此次已补齐。

在线精确修复8条（compact=1）：四个legacy perpetual查询改prohibited，instrumentKind future；四个真实future查询用官方base/quote/到期/保证金，独立crypto:binance:future:<rawSymbol>身份与比较键、linear_futures、ambiguous，不与永续或不同期限共享比较身份。仅更改具体市场，不全局改BTC/ETH类、别名或单位。严格写前比较及即时回读零差异。最后写入01:09:42Z后，成功自然同步01:14:37Z→01:14:38Z，再回验1,165条新旧规则零错误、零覆盖丢失；BTC/ETH资产显示视图亦回读成功，但显示资产名不是具体合约身份。证据见binance-dated-boundary-check、online-write、final-online-verification；新增future键在binance-native-future-supplement-verification。

根因与防复发：采集原来把所有Binance futures-info条目填成perp，丢失deliveryDate/marginAsset；curation的future入口和日期隔离只支持Bybit；无精确规则时解析器仍把underscore日期当货币。现在季度与_DELIVERING变体保留native contractType、expiry、settlement并按future导入；复用日期合约隔离、原生证据优先级、冲突禁用、非prune旧记录隔离和禁止从通用token别名回填季度身份。无精确规则的Binance原生日期语法返回unresolved，后缀只作拒绝/隔离，不能推算expiry/quote。仅USD-M官方端点可归一化为linear_futures；COIN-M同名季度枚举保持未核实隔离，不能借用线性/保证金单位。普通PERPETUAL不受影响，quote/expiry/settle来自官方字段。schema15使部署后拒绝旧动态生成层；源码现在同时支持Binance/Bybit，不靠只修生成缓存。只读审计增加--official-binance及official_quarterly_future_boundary_review线索，后续每日需补入官方future类型、去重与核对。

当前官方覆盖Extended416、gTrade493全部类别、GRVT184、GMX126、Gate1080期货、Bybit含40条当前交割、Hyperliquid全部DEX、Bitget现货/永续、Binance USD-M、QFEX、Variational、Aster、Lighter、Backpack、OKX SWAP。Binance目录本身没有新增，当前类型703 PERPETUAL、235 TRADIFI_PERPETUAL、2 CURRENT_QUARTER、2 NEXT_QUARTER；此次修正既有误分类，不是新上市。额外核对身份字段：Binance underlying/base/quote/margin/delivery、OKX ctType/ctVal/ctMult/ctValCcy/settle/instCategory、Lighter ID/deployer/multiplier/settlement、Aster subtype/margin/status、QFEX margin/category、Extended/GRVT身份类别，均未见今日字段变更。全量来源URL/时间见fetch-manifest、supplemental-fetches。

主并集比昨日新增144键：143个Gate杠杆现货均unknown/ambiguous，io:META为官方inactive占位prohibited，不自动赋予可比资格。无新增到期或外部部署比较告警；此前STRC/URA及外部ASELS/TUR30人工边界继续持久。COPPER/ICE/历史链与包装证据待核实项保留，无反复写入。健康样本不是本日覆盖范围替代。

线上发现19,543条，比昨日20,130少587：gTrade/Extended源行减少、GMX从聚合inventory缺失；当天官方目录已补查，不用旧快照冒充实时；缺少collector失败遥测，不能精确归因。主平台数20包括官方补充，不能称19个当前发现平台均独立完整。Lighter-Robinhood71个consumer键检查，独立原生目录仍未核实；示例api.example.com反馈源不可作为实时反馈。线上仍4efa310/schema10，本地schema15未部署。

验证：完整Go套件排除既有TestHandleDiscoveryLookupAggregatesAllDiscoverySources后通过；该用例单独回验仍GroupCount3 vs2，未修改其测试/聚合器。覆盖采集到解析、永续与不同到期隔离、遗留perp禁用、未知日期拒绝、native expiry冲突、历史条目非prune隔离；vet、构建通过，Python9项通过。21,886行本地服务回放wall605ms、CPU754ms、读p50 583ns/p99 1084ns、heap190.4MB。不是生产性能改善对照。高频解析新增固定长度后缀检查，无每请求regex或全序列化；采集/生成周期补4条原生季度记录，未改刷新/丢弃/陈旧/队列策略，无JS事件循环变化。线上尚未部署，不宣称生产CPU变化。

审计记录保留各条读取时间和修复前flags，修复后处置引用最终回读。仅用官方deliveryDate，未从日期后缀猜到期；未推送、未部署、未重启。原工作区无跟踪文件修改，已有日志和状态保留。
