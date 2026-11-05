# 2026-11-05 北京时间巡检

覆盖线上 discovery 与 registry overrides 主并集 20,355 个精确市场键，20个平台；官方 Aster/Lighter/Backpack 原生目录补充40键，总计 20,395 条逐条 disposition。输入证据21,917条；每批250、最多2并发。另存6条 Aster 缺失 contractType 的未确定元数据，未当作已解析合约。resolved或无flags不代表独立核实可自动交易。

当日官方覆盖 Extended416、gTrade493全部类别、GRVT184、GMX126、Gate期货、Bybit线性及40条交割、Hyperliquid core及全部DEX、Bitget现货/永续、Binance、QFEX refdata、Variational、Aster、Lighter、Backpack。原始来源URL/时间见fetch-manifest、supplemental-fetches与各专题文件。QFEX最初错误 /assets 端点返回401，后按项目配置 /refdata 成功抓取并保留纠正证据；前者不是正式数据源故障。没有使用历史快照冒充今日成功抓取。

实质修复89条，按具体市场写入compact=1：

- Bitget本次官方目录新观察到86个isReality=yes、symbolType=stock现货，保留股票分类，明确tokenized_equity/reality_stock_token和equity:bitget:reality:<raw>身份，ambiguous，不把包装代币直接等同股票/永续。不改全局别名或资产类别。目录差异不代表首次上市日期。
- Variational TBT：官方名称ProShares - UltraShort 20+ Year Treasury，ProShares官网确认ETF，crypto误类改rwa_stock；SQQQ亦据原生名称和ProShares产品资料保留ETF身份。两者使用各自平台永续身份、ambiguous；不与国债/QQQ合并，不把-2x/-3x日目标写成单位换算乘数。
- Bitget FISVUSDT：官方isRwa YES、perpetual、USDT保证金；保留现有股票类，平台永续身份ambiguous，不仅凭ticker推定发行人或跨平台份额。

写入前严格比较、即时回读通过。最终1,137条新旧规则在晚于最后写入的成功自然同步后回验，零差异、零覆盖丢失；见online-write.json、final-online-verification.json。另对TBT/SQQQ/FISV/CAT/GOOG/DFEN/GGLL/UVIX回读资产显示视图，显示名不是跨平台唯一身份。官方ETF证据：[TBT](https://www.proshares.com/our-etfs/leveraged-and-inverse/tbt)、[SQQQ](https://www.proshares.com/our-etfs/leveraged-and-inverse/sqqq)。

主并集新增528键：Gate430个杠杆现货均维持unknown/ambiguous，不推定换算；Bitget88、Variational2、Bybit1、Lighter1、OKX6。新增Bitget87现货含86包装股票及1个crypto KOR，后者不套股票规则。Bitget5个产品状态online变limit_open，仅记录状态，不推断下架；RINDV从本次官方目录消失，单次缺席不足证明下架。Extended HIVE/USD显示变体unknown/ambiguous与官方HIVE-USD crypto分类有差异；原生合约是Hive crypto，显示变体来源/身份证据未独立核实，保留隔离而不扩大可比资格。Hyperliquid官方身份字段较昨日无变。

线上采集19,588条，比昨日+339；health读取时上一次同步17,944条与稍后inventory19,588不同，反映不同批次且旧版不暴露部分源失败ID，不据此断言源无缺失。Lighter-Robinhood67条consumer键已检查，独立原生目录仍未核实。示例反馈域名api.example.com不是可用消费者反馈源。线上仍4efa310/schema10，遥测和本地schema12尚待部署。

系统性防护：仅对官方Bitget现货股票元数据保留isReality标记；生成时原生包装证据优先于泛化股票标签，以rawSymbol确定独立包装身份，不再只靠剥离r前缀共享股票身份。限定原生来源、stock、spot、Reality条件；IPO Prime和非股票不误套。只在股票采集记录分配额外元数据；无高频消息路径新增队列/序列化/全表重算。生成版本12以部署时拒绝旧动态层。测试覆盖双输入顺序及不可信来源、非包装、永续、crypto反例。

验证：隔离保留用户既有aggregator两文件的提交视图完整Go测试/vet/构建通过，Python6项通过；真实工作区仍有昨日已知分组测试3 vs 2差异，未修改或隐藏。完整21,917行本地服务回放：同步wall1.067s、CPU1.034s，读p50 833ns/p99 3875ns，heap169.7MB。单次本地测量非生产优化对照；未改变刷新频率、丢弃或陈旧策略，不涉及JS事件循环。待部署后观察线上CPU与同步新鲜度。

逐市场结果见dispositions.jsonl，主audit.json保留写入前结果时间；修复后的处置引用final-online-verification.json。未推送、未部署、未重启；用户既有工作区改动保留。
