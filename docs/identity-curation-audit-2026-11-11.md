# 2026-11-11 北京时间巡检

主并集20,430键/20平台，Aster/Lighter/Backpack/Extended SPOT原生补充44键，总计20,474条逐市场disposition；主输入21,957条。另存6条Aster空contractType元数据，不当成解析成功。每批250、最多2并发，API错误0。resolved/无flags不表示独立认证交易可比性。

线上14条精确修复（compact=1）：Lighter NOK/AXTI/USDHKD/POPMART/BYD/SOXS由官方active转inactive，原eligible改venue/raw专属inactive身份、prohibited；前5个泛化crypto标签缺少发行人/FX证据，改unknown，SOXS沿用原stock类别但不认证单位/杠杆。保留现有USDC结算，不从零asset IDs或native multiplier推断新结算/对冲比例。Aster APMUSDT/DELTAUSDT官方SETTLING，原eligible改专属settling身份、prohibited，使用官方base/quote/margin，不把永续哨兵deliveryDate当真实expiry。Variational TEAM/GTLB官方名称Atlassian Corporation/GitLab Inc.，去掉错误crypto身份，按独立股票永续产品ambiguous，未推断股类或跨平台单位。Bitget新RQUITUSDT/RTXCPUSDT和旧RACHVUSDT/RTCBIUSDT分别保存Reality tokenized_equity身份、ambiguous；对应官方ISIN相同，但不自动合并或推断转换比例。旧符号仅使用Nov10存档证明此前包装身份，不称其当前仍在目录或断言下架时间。

写前完整比较和即时回读错误0。最后写入2026-11-11T01:08:01.830011+00:00后，成功自然同步2026-11-11T01:08:37.058823231Z→2026-11-11T01:08:39.021973407Z；1181条新旧规则错误0、覆盖丢失0，相关10个assets视图回读成功。资产显示不替代具体产品身份边界。原始修复前/请求/响应/回读见repair-plan、online-write和final-online-verification，精确证据已固化审核层。已有生成规则覆盖inactive/settling/Reality隔离，今日未改高频运行时逻辑。

主并集新增7键：Bitget2个新wrapper、Hyperliquid @1189、OKX AERO-USDT/XORCL-USDC、Variational2股票；无主键消失。@1189原生token index1333、tokenId0x5f0c311fab64ce729122ed14d39c7332、name XMR2、fullName XMR - Beluga；没有与XMR合并，不因名称认证跨链/跨平台等价。OKX新品unknown/ambiguous，未猜包装单位。Extended新EURCSPOT-USD为SPOT不是perpetual；补查全部4个原生SPOT，其中3个既有目录先前仅perp补查未覆盖。EURC、wETH、cirBTC等uiName与原生assetName/quote有差异，保留原始source冲突；4键现有均ambiguous，USDTSPOT-USD官方DELISTED并继续禁止自动比较，未猜USDC=USD或原生包装可比。补充键含这4条，不能把全部新增覆盖都称新上市。

Aster IBMUSDT和STRCUSDT从PRE_SETTLE回TRADING，Hyper io:SKHX撤inactive标记，保留原禁用等待身份/单位复核。Gate ROKU_USDT从当前官方目录缺失，但仅缺失不证明下架，原ambiguous隔离不变。Bitget65条ISIN补充、RBRKB fullname变空，均保留官方前后证据，既有Reality边界不反向覆盖。别名/乘数/ADR/普通股/期货到期与同名股票crypto边界保留。

官方当日覆盖Extended416 perpetual+4 spot、gTrade493所有组别、GRVT184、GMX126、Gate futures、Bybit完整linear分页、Hyper全部DEX/spot、Bitget现货/永续、Binance USD-M、QFEX、Variational、Aster、Lighter、Backpack、OKX SWAP/SPOT，来源及请求时间见fetch-manifest/supplemental-fetches。源抓取失败0不等于所有后端collector健康：聚合inventory19618行vs19774，GMX仍缺失，已用今日官方目录补126行；当前线上没有失败collector遥测，不能精确归因少行。Lighter-Robinhood71 consumer键有结果，独立provider原生目录仍未核实。

保存逐市场snapshot读取时间和flags；14条修复最终disposition用同步后回读替代，不把批量审计并发中的旧结果冒充修复后状态。比较昨日snapshot还发现若干Hyper源分类从unknown/crypto变成stock/ambiguous；作为source字段变化记录，不作为独立单位认证或新手工修复计数，不全局重写。Binance4季度future已在主并集，未重复补充。

验证identity/curation/bootstrap测试通过，嵌入数据服务构建和git diff --check通过；只改14条持久规则和证据，未改变刷新频率/丢弃/队列或高频路径，不宣称生产CPU/事件循环改善。既有server聚合GroupCount3 vs2历史失败未改，见Nov8记录。线上仍4efa310/schema10，本地schema15待部署。未推送、部署或重启，原日志和未跟踪状态保留。
