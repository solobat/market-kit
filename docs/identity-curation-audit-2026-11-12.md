# 2026-11-12 北京时间巡检

主并集20,319键/20平台，官方Aster/Lighter/Backpack/Extended SPOT额外44键，合计20,363条独立逐市场处置；主输入22,487条证据。另存6条Aster空contractType元数据，不计解析成功。每批250、最多2并发，API错误0。resolved/无flags不表示独立核实可自动交易。

本日共145条线上精确修复：142个Binance非交易合约和3个Arcus股票永续产品。

本次发现系统性遗漏：Binance官方PERPETUAL/TRADIFI_PERPETUAL的PENDING_TRADING/SETTLING未进入比较资格判断。完整官方目录中有137条SETTLING、5条PENDING_TRADING；142条原141 eligible、1 ambiguous，全部按原生状态改prohibited与venue/raw专属settling/prelaunch身份。4个当日新增为FRHCUSDT/NUUSDT/HYPEUSDC/TRXUSDC，均PENDING_TRADING；其余是历史覆盖遗漏，不是今日新下架。保留原有显示、倍率及单位别名，官方quote/margin独立读取，不从日期哨兵、名字或价格猜expiry或单位；不全局改股票/crypto资产。全部当前非交易状态有binance-all-nontrading-check/repair-plan逐项证据，精确写前比较、compact=1批写、即时回读零差异。

最后写入2026-11-12T01:14:04.944561+00:00后，成功自然同步2026-11-12T01:14:37.059190019Z→2026-11-12T01:14:39.073674745Z；1326条新旧规则零错误、无覆盖丢失，相关资产视图回读成功。resolve为具体产品身份依据，assets显示不能证明合约相同单位。每条修复处置用最终回读覆盖，并保留snapshot flags/时间；审计脚本重跑仅给同指纹缓存附加新状态flags，不把旧读取冒充写后实时回验。

防复发代码：bootstrap保存原生nativeTradingStatus、contractType、marginAsset和官方USD-M来源。生成层只接受精确USD-M端点、PERPETUAL/TRADIFI_PERPETUAL的明确PRE_TRADING/PENDING_TRADING/SETTLING排除证据，输出专属身份与prohibited；非交易原生证据优先于弱consumer live标签，来源顺序无关。不由未知状态、源标签或COIN-M同名枚举推断；普通TRADING产品比较行为不变。冲突保证金保持未知，解析器对两种unverified非交易合约不从quote自动回填。schema16使部署后不复用旧动态生成语义；只读审计--official-binance现在检查全部原生永续非交易状态，季度类型/到期核对仍独立，不把普通带日期名字当future。

新主键30，缺失旧键141（均Gate），主键净少111。聚合inventory20147行vs19618，不能当作新增529个标的或已证明141个下架。保留采集来源差异，不自动移除历史注册规则。Arcus5、Binance4、Bitget MTX、Gate DANGOTE/MIHAO、Hyper io:INTC、OKX11、Ondoperps6等新增各有结果。Bitget MTX官方crypto/isReality=no，不按股票包装猜测。Gate新合约quanto_multiplier10/100是合约价值字段，不自动当价格/身份倍率；仍unknown/ambiguous。Hyper io:INTC已有prohibited边界不自动放行。OKX新现货unknown/ambiguous或unresolved，保留原生raw。Ondoperps新中文公司/未上市产品身份与单位仅consumer证据，未据名称全局合并，保留未独立核实说明。Arcus官方补抓结果单独见arcus-fetch，不用失败或旧目录冒充当前。Arcus全部86个原生目录键与主并集对应，股票新品CBRS/WDC/NFLX官方EQUITIES及公司名称明确，但未独立证明股类和合约单位，故用equity:arcus:perpetual:<raw>身份ambiguous；维持既有结算字段而不新认证保证金单位，不作全局发行人别名。142个Binance批次和3个Arcus批次分别保存原始before/response/after，组合online-write仅为最终manifest。已有Lighter inactive、Aster结算/恢复、Reality新旧包装及ADR/普通股等边界不变。

QFEX IGV-USD官方category由INDEX改EQUITY，现有线上已经rwa_stock；无需重复写。quote USD与margin_asset USDC继续区分，类别更新不构成新跨平台单位证明。Bybit4STOCK/AGI/HAJIMI原生LinearPerpetual新增未来deliveryTime，记录未来结算/下架时间线索；当前status未变，不提前当已到期交割产品或误认4STOCK为股票。Gate FLYBRAIN当前官方目录缺失不独立证明下架。Extended BILL和gTrade DAR的弱stock提示不覆盖既有已核实精确crypto隔离；HIVE/COPPER待核实项保留。

当前官方覆盖Extended416 perpetual+4 spot、gTrade493所有类别、GRVT184、GMX126，及Binance USD-M、Bybit完整linear分页、Gate futures、Hyper全部DEX与spot、Bitget SPOT/futures、Aster、Lighter、Backpack、QFEX、Variational、OKX SWAP/SPOT。来源/时间/请求失败分别见fetch-manifest、supplemental-fetches、各抓取记录。当前GMX聚合源已出现但重复base/quote来源冲突保留，不能将primary source数量与独立核实范围混同。Lighter-Robinhood71 consumer键核查，独立provider原生目录仍缺口；线上尚无collector失败遥测，不宣称全部采集器健康。

验证：完整Go套件排除既有TestHandleDiscoveryLookupAggregatesAllDiscoverySources后通过；该用例单独回验仍GroupCount3 vs2，未改其行为/断言。原生状态优先级、弱来源两种顺序、普通交易保留、官方源边界、保证金冲突与不猜expiry有回归覆盖；vet、服务构建通过，Python10项通过，diff检查通过。22629行本地回放（22487主证据+142原生状态证据）：sync wall753.8ms、CPU878.6ms、读p50 1292ns/p99 29958ns、heap214.7MiB。该轮与完整测试并行，数据/环境不同于昨日，没有同输入生产前后对照，不能宣称生产CPU或尾延迟改善；保留高p99观测，不隐瞒突发读取延迟。新增状态决策只在现有采集/生成周期，Resolve只新增两个固定contractType分支以抑制错误回填，无新每tick任务、队列、序列化或刷新/丢弃/陈旧策略变化，无JS事件循环改动。

线上仍4efa310/schema10，本地schema16和持久审核修复待推送部署；线上人工修复已持久。未推送、部署、重启或暂停同步，保留原未跟踪日志及巡检状态。
