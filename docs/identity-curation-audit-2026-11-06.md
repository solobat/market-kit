# 2026-11-06 北京时间巡检

覆盖主并集19,955个市场键/20个平台，补充原生Aster/Lighter/Backpack目录40键，总计19,995条逐条disposition；另存6条Aster缺失contractType元数据，未冒充解析结果。输入20,116条当日证据，每批250、最多2并发。保留chain/source冲突。resolved及无flags不等于独立核实交易可比性。

在线精确修复15条（compact=1）：Bitget12个新观察到的股票永续（BHARTIARTL/HDFCBANK/HINDUNILVR/ICICIBANK/INFY/ITC/KMB/LT/MS/RELIANCE/SBIN/TCS），原生isRwa YES并结合官方股票公告，修正9条unknown，其他保留股票类；全部独立平台产品ambiguous，不推断印度普通股/ADR、INR参考/USDT结算或跨平台单位换算。Variational MARA原生全名MARA Holdings, Inc.纠正crypto为股票；MCD原生McDonald's Corporation保留股票类；两者独立平台永续ambiguous。Aster STRCUSDT原生underlyingSubType STOCK纠正crypto类，官方状态TRADING→PRE_SETTLE，保守禁用比较，标记pre_settle_perpetual；不称已结算/已下架，不推断历史日期、优先股现金流或份额等价。

即时回读无差异；15条与此前修复共1,152条，在00:59:40Z最后写入之后的成功自动同步01:00:37Z→01:00:39Z回验，零错误、零覆盖丢失。15个对应资产显示视图另行回读，均成功；显示名/全局资产视图不是跨平台唯一join依据。核对字段含类别、assetId、underlyingId、comparisonKey/status、kind、contractType、settlement、expiry和乘数。修复前后及证据见online-write、repair-plan、final-online-verification、新资产回读。

当前官方目录覆盖Extended416、gTrade493全部组、GRVT184、GMX126、Gate1080期货、Bybit含40条交割、Hyperliquid core及全部DEX、Bitget现货/永续、Binance、QFEX、Variational、Aster、Lighter、Backpack。具体URL/时间见fetch-manifest/supplemental-fetches及专题文件。Gate新增10个股票永续已由现有原生规则保持独立产品ambiguous，无重复写入；新xyz:STX官方inactive占位保留prohibited，不按同名STX合并crypto。QFEX FTNT INACTIVE→ACTIVE仅记录状态，不修改既有可比规则。Bitget4个目录产品消失、4个limit_open→online不证明下架/换股。新增主键30个，昨日430个Gate杠杆现货键本日不在当前并集，保留历史记录而不以旧清单冒充实时核查。

本日inventory17,777条，昨日19,588（-1,811）；Aster/Lighter/Variational/Lighter-Robinhood未出现在当前源，前三者用官方目录补查，RH独立目录仍未核实。当前registry中的历史键全部进入本次解析；但无法证明独立RH目录完整或获取实时交易状态。线上仍4efa310/schema10，失败源字段未部署，空cacheError不证明无部分失败；采集健康缺口如实保留。消费者反馈地址api.example.com是示例不可作为实时证据。

代码修复系统原因：Bitget永续采集此前丢失isRwa、symbolType及支持的保证金币种。现在仅在官方RWA记录保存这些字段；生成时native RWA证据优先于泛化标签，平台产品独立ambiguous，类别unknown直到外部审核明确股票/商品/指数，不再猜crypto或共享ticker。多保证金/缺失唯一保证金及相互冲突时不回填报价币为结算币，使用rwa_perpetual_unverified守住未知边界；原生RWA不产生全局crypto别名。精确审核规则提供此次确认股票类。schema13使部署后拒绝旧动态生成层。

回归覆盖采集到解析、两种输入顺序、非原生来源/非RWA/现货/交割反例、冲突和多保证金。完整Go测试中既有服务器分组测试3 vs2差异仍在（用户已提交聚合器变更5b68a02），其余全套通过；未修改该测试或聚合器。vet、构建、Python6项通过。最终本地20,116行服务回放wall623ms、CPU753ms、读p50 625ns/p99 2125ns、heap212MB；单次本地结果不是生产性能改善对照，未部署无法报告线上CPU变化。工作在采集/生成周期，无新高频队列/全表事件重算；未改刷新/丢弃/陈旧策略，无JS事件循环变化。

官方公告证据：[印度股票10合约](https://www.bitgetapp.com/asia/news/detail/12560606110681)、[MS/KMB](https://www.bitgetapps.com/support/articles/12560603899780)。使用官方域搜索摘要；完整页面打开失败，未声称成功抓取全文。原生API元数据另行成功抓取。

逐市场处置见dispositions.jsonl；audit.json保留各条读取时间，修复行在dispositions引用最终验证。沿用COPPER、ICE、历史包装和碰撞待核查项，不重复改写已受阻规则。未推送、未部署、未重启；原工作区无跟踪文件修改，日志和已有状态保留。
