# 2026-11-10 北京时间巡检

覆盖主并集20,423键/20平台，Aster/Lighter/Backpack官方原生补充40键，总计20,463条逐市场处置；主输入22,114条。另存6条Aster缺失contractType元数据，不计作解析成功。每批250、最多2并发，无API错误。resolved/无flags不等于独立认证可比较或可交易。

线上精确修复2条：Aster IBMUSDT官方status由TRADING转PRE_SETTLE，原eligible通用IBM身份改为venue/raw专属pre_settle身份、prohibited，保留官方USDT结算，不用永续deliveryDate哨兵作为真实到期。STRCUSDT同样PRE_SETTLE但已有禁用规则，未重复写。Bitget新增RBRKBUSDT明确isReality=yes/symbolType=stock/fullName=Berkshire Hathaway-B，原通用BRKB eligible改为equity:bitget:reality:rbrkbusdt独立tokenized_equity/reality_stock_token、ambiguous；不推断现金股换算或跨平台同单位，不全局修改BRKB别名类别。写前完整比较、compact=1写入、即时回读零差异。

最后写入2026-11-10T01:02:54.301080+00:00后，新成功自然同步2026-11-10T01:03:37.058882925Z→2026-11-10T01:03:39.165274302Z，1167条新旧规则错误0，人工修复覆盖丢失0；IBM/BRKB/ANTHROPIC/XYZ100资产视图亦回读，资产显示不是具体产品比较依据。修复前后及来源见repair-plan/online-write/final-online-verification。持久审核层与官方证据已更新；防复发复用已有Aster状态隔离和Bitget Reality生成规则，无需修改热路径。

今日主并集新增12键：RBRKBUSDT、Gate牛逼_USDT、Hyper xyz:PANW及9个OKX X前缀现货；无主键消失。Gate新品无独立身份映射，保持unknown/ambiguous；9个OKX现货已从官方SPOT目录逐项核对，但未据名字推断包装发行人、杠杆或单位，仍unknown/ambiguous。Hyper xyz:PANW在原生目录出现，已有禁用边界保留；xyz:MUU/xyz:TTWO撤除inactive标记也不自动放行，不把MUU当普通MU。所有新增条目均有disposition。

官方全目录覆盖Extended416、gTrade493所有组别、GRVT184、GMX126；另含Gate、Bybit分页、Hyper所有DEX及spot、Bitget现货/永续、Binance USD-M、Aster、Lighter、Backpack、QFEX、Variational、OKX SWAP/SPOT。官方身份字段其余无变化，保留每次请求URL/时间。Binance4个季度future已在主并集，不重复计数。已有ADR/普通股、1000乘数、同名股票/crypto、外部部署/占位、包装资产边界和未核实项保留。

聚合inventory19,774行vs昨日19,467；来源组成变化，不能当成307个新标的。Bitget新增bootstrap/slipstream双源base表示冲突大量增加，例如AAGIY与rAAGIY；保存原始双源证据，不让通用crypto/base标签覆盖官方stock/Reality身份。主flags为修复前读取，处置文件用最终回读覆盖2条修复并保留preRepairFlags。GMX缺失于聚合，已补当前官方126条；缺少线上collector失败遥测，不宣称全源采集健康。Lighter-Robinhood71 consumer键已检查，独立provider原生目录仍未核实。

验证identity/curation/bootstrap全部通过，服务构建通过，git diff --check通过。仅更改2条持久规则与证据，无新运行时逻辑，不重复生产CPU/事件循环测试或宣称性能变化；刷新、丢弃和陈旧策略不变。已知server聚合GroupCount3 vs2历史失败未改，详见11月8日。线上仍4efa310/schema10，本地schema15防复发代码待用户部署；未推送、部署或重启，保留原日志和未跟踪状态。
