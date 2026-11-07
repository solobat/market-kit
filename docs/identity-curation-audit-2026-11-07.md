# 2026-11-07 北京时间巡检

主并集20,263键/20个平台，原生目录额外40键，总计20,303条逐市场disposition；6条Aster缺少contractType只作未确定元数据记录。输入22,469条证据，每批250、最多2并发，保留chain/source差异。无flags或resolved不等于独立核实可交易。主audit.json保留实际各条读取时间及修复前快照，dispositions中的修复行引用最终回读。

在线精确修复21条，三批compact=1写入，严格写前比较、写后回读：

- Bybit16个已到期future键：BTC/DOGE/ETH/HYPE/MNT/SOL/XAUT/XRP，各有06NOV26和30OCT26到期合约。Nov6与Oct30官方存档deliveryTime分别1793952000000、1793347200000，与当前注册expiry一致，现均已过期。保持精确交割身份、期限、保证金与乘数，只把comparisonStatus从ambiguous改为prohibited。存档证明历史合约规格，不冒充实时交易目录。新增到期巡检还找出了8个Oct30遗留条目，已全部处理；未称为今天新下架。
- Lighter ASELS与TUR30：官方market_deployer=external，精确ID8286/4647，原先仅凭名称为crypto eligible；现在unknown、market-ID/raw-symbol限定产品、ambiguous。没有官方发行人/资产类/单位证明，不猜股票或指数。原USDC结算字段保留，不在本次独立认证其单位。昨日inactive URA ID150与ASELS不同，不能按改名合并；URA手工prohibited保留。
- OKX EIDO/EWZS/XLV-USDT-SWAP：官方instCategory3=Stocks、ctType linear、ctVal/ctMult1、settleCcy USDT。保留股票类，使用各自平台永续产品ID、linear_swap、ambiguous；不把面值1当作跨平台份额/包装等价证明，不修改全局类或别名。

截至最终批次01:11:29Z之后，成功同步01:11:37Z→01:11:38Z后回验1,157条新旧规则，零差异、零覆盖丢失。21条修改中16条替换既有检查键，新增5个检查键；13个对应资产显示名另行回读成功。全部分批证据见first-online-write、okx-online-write、legacy-expiry-online-write，online-write仅为合并期望规则清单，最终解析见final-online-verification。

官方覆盖Extended416、gTrade493全部组、GRVT184、GMX126、Gate1080期货、Bybit线性含40条当前交割、Hyperliquid core与全部DEX、Bitget现货/永续、Binance、QFEX、Variational、Aster、Lighter、Backpack；另抓OKX SWAP官方完整目录。URL与时间见fetch-manifest/supplemental-fetches/okx-fetch。今日新增主键308，其中288个Gate杠杆现货均unknown/ambiguous，保留不推断倍率；Bybit新3个股票永续和8个交割已由原生规则隔离，新io:MU inactive占位prohibited，无重复写入。RH新增ENA/PUMP/WLD/XPL保留USDG边界，独立目录及产品规格未核实，不把mainnet Lighter当RH证据。Variational/QFEX身份字段无新增变化。

STRCUSDT官方PRE_SETTLE→TRADING，仅记录恢复状态，保留既有prohibited直到发行人/单位复核，不自动恢复比较。REFV恢复online、Bitget5个现货从目录消失，仅作目录差异，不推断换股、下架日期。inventory20,130条，比昨日17,777增加2,353，昨日缺失平台重返当前发现清单，但线上仍4efa310/schema10，没有部分失败字段，空cacheError不能证明全部配置源完整。Lighter-Robinhood独立原生目录仍是覆盖缺口，consumer样本反馈仍为api.example.com示例地址。

系统性代码：OKX采集保留原生股票类别、正反向合约类型、保证金币种、原始面值/面值币种/合约乘数字符串；仅为股票SWAP附加元数据。复用原生股票产品隔离规则，优先官方类别，默认平台产品ambiguous，避免未来新品由ticker共享股票身份；正向/反向分别linear_swap/inverse_swap，缺失结算用_unverified守住未知，不能从报价币回填。面值字段不自动映射价格别名倍率；同级原生面值/面值币种/乘数/保证金证据冲突时保守unknown/ambiguous，stock_swap_unverified不回填报价币。生成schema14，部署后拒绝旧动态生成层。只读巡检新增外部部署身份线索和明确expiryAtMs过期线索，不从显示symbol解析日期，不自动交易/改类。

验证：完整Go套件排除既有TestHandleDiscoveryLookupAggregatesAllDiscoverySources后通过；该测试单独回验仍是GroupCount3 vs2的已有差异，未修改聚合器或该测试。最终8条遗留规则补充后identity/curation再次通过、最终构建成功；vet通过；Python8项通过。覆盖native类别优先级、正反向/缺失结算、面值与价格倍率分离、外部ID非别名和到期边界。22,469行本地服务回放wall1.262s、CPU1.112s、读p50 833ns/p99 2709ns、heap191.5MB；单次本地结果不是线上性能改善对照。未改高频输入队列、刷新频率、丢弃/陈旧策略；无JS事件循环变更；线上尚未部署，不能宣称生产CPU已改善。

官方产品与类别依据：[OKX接口指南](https://www.okx.com/docs-v5/en/)、[官方SWAP目录](https://www.okx.com/api/v5/public/instruments?instType=SWAP)、[Lighter原生目录](https://mainnet.zklighter.elliot.ai/api/v1/orderBooks)。没有用第三方ASELS名称/价格线索自动认定发行人或资产类。

沿用COPPER、ICE、历史链/包装与单位待核查；不重复写入已知被覆盖的同类规则。未推送、未部署、未重启。原工作区无跟踪文件改动，已有日志和状态保留。
