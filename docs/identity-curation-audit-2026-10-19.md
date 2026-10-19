# market-kit 每日巡检：2026-10-19

## 覆盖与结果

检查20个平台、19,674个唯一市场。起始线上discovery为19601行，cachedAt=2026-10-19T00:55:24.059712269Z；合入Extended/gTrade/GRVT/GMX官方目录及Gate完整合约目录后共21850行来源记录。对完整并集执行79批在线解析，每批250、最多2并发，无API批次失败。各平台JSONL逐条记录disposition、来源、身份字段及尚未独立核实的边界。静态清单与官方抓取/解析时间逐条留存，不称为持续实时监控。

今日线上精确修正/收紧123条Gate规则：96条官方stocks合约的漏分类或crypto-ID混用；4条金属合约类别/ID；18条指数独立产品边界；4条外汇产品边界；BEN_USDT因stock类别与crypto-ID冲突且发行人证据不足，保留unknown/ambiguous独立产品。不是123条已证明同类经济资产，也没有据此开放自动跨平台交易。

股票和金属规则使用venue范围产品ID，comparison保留ambiguous。官方indices包含波动率等非股票指数，故保留unknown类别、index_perpetual和独立index ID；不一律当作股票。Gate明确把USDC_USDT标为forex，现与稳定币USDC分离，四条FX均prohibited。IAU/SLV仍保留ETF/股票类，PAXG/XAUT仍保留商品背书敞口，没有按metals/crypto设施分组直接误改其产品身份。

Gate官方1042个合约完整补查，其中413条stocks全部核对；修复后股票类别及ID命名空间无剩余差异。Bitget官方2915个股票类市场也全部核对，无新增类别差异。官方类别、原始symbol、合约类型、in_delisting、quanto_multiplier均保留证据。quanto_multiplier表示每张合约对应的底层数量，不复制成market-kit的价格/符号倍率；新规则原生价格单位保持1，不能据此跳过消费者的张数转币数处理。

## 既有修复与同步验证

线上已部署46b2cc0（构建2026-10-18）。历史108条修复回读通过，38条gTrade多抵押外汇的settlementAsset已保持未知，昨日部署阻塞关闭。Anthropic、XYZ100、IPO Prime等既有产品边界保留。

今日123条新增规则连同历史108条，共231条，在本次写入后启动并成功完成的自动同步后逐字段回读，无差异。同步开始2026-10-19T02:45:17.874558264Z，完成2026-10-19T02:45:19.889607857Z；回读时间2026-10-19T02:45:44.064605+00:00。相关assets也已读取，不批量修改全局同名资产类别。线上manualOverrideCount=231。

## 新增、缺失与保留线索

相比昨日并集新增3个覆盖key：Gate CLAUDIA_USDT、Gate PAPER_USDT及Hyperliquid现货@1117。不是三个已经确认的新上市事件。CLAUDIA/PAPER官方合约存在，但empty contract_type不是crypto类别证据；CLAUDIA币种查询返回400，合约上市公告未成功取得，不能将同名Trenches Solana token直接并入。两者保留unknown/ambiguous，未猜测链或项目。

官方spotMeta确认@1117引用token index 1273的BYQYS（tokenId 0x09c9c5971dd28c23a28b1e39e794fcce），quote为index0的USDC；isCanonical=false。token索引稀疏，使用index映射而非数组下标。原生资产目录核对不等于已证明外部桥接、其他平台同名资产或价格单位可比，本次未修改其规则。

昨日129个Gate杠杆现货key今日不在并集中，全部带倍数L/S后缀；逐条名单保留，不把来源缺失称为已确认下架，也未删除规则或与普通币合并。Bitget官方400个记录仅状态online→limit_open，symbol/type/ISIN/base/quote未变；采集器已有paused映射，不把状态变化当成新资产。

gTrade493对、12分组（包括股票/指数/商品/FX）全部补查；Extended404条perpetual、GRVT184条、GMX126条可解析池证据保留。GMX同名不同池/链不自动合并。DAR的泛化stock来源冲突由gTrade官方altcoins/coin证据解释。旧历史身份、包装资产和unknown-ID线索继续保留，不把flags当作确定错误或resolved当作独立验证。

## 代码、性能及待执行

修复Gate采集器读取contract_type，并为官方typed产品恢复原生base/quote；空类型不猜crypto。股票、金属和ETF保留区别；指数和FX在生成层阻止借用crypto/稳定币身份。全部规则与官方证据嵌入reviewed数据，部署后无需重复在线写入。

Go全量race、vet、Linux amd64构建通过；21850行输入跨两轮同步保留1867条reviewed规则。同一官方补充输入对比46b2cc0，仅22条Gate指数/FX生成记录改变，无其他平台变化、删除或新增碰撞；采集器新增读取字段由fixture另行验证。

本机重放样本sync wall≈0.852s、CPU≈0.990s、reader p99≈4.83μs，非生产性能承诺。改动在原有低频采集/生成流程，无新增运行时HTTP请求，没有改变ticker路径、同步频率、队列或丢弃语义；分类输入更完整可能增加低频生成工作量。前端未改动，未重复跑前端构建。新代码本地提交待推送/部署，未擅自发布或重启。

主目录补查均成功；CLAUDIA发行人/链匹配仍有上述覆盖缺口。consumer反馈源仍配置api.example.com占位URL，未宣称检查真实反馈；bootstrap单collector失败可观测性问题仍保留待改进。

## 证据

fetch-manifest.json、gate-futures-fetch.json、new-official-manifest.json记录URL和抓取时间；inventory/registry/audit-inventory/audit保存输入与初始诊断；各平台JSONL保存最终逐市场disposition；gate-repair-plan.json、gate-online-write.json、final-online-verification.json保存写前后与231条跨同步回读；generated-differential.json保存同输入比较。无凭据保存。

主要官方来源：[Gate完整合约目录](https://api.gateio.ws/api/v4/futures/usdt/contracts)、[Hyperliquid info](https://api.hyperliquid.xyz/info)、[Extended](https://api.starknet.extended.exchange/api/v1/info/markets)、[gTrade](https://backend-arbitrum.gains.trade/trading-variables)、[GRVT](https://market-data.grvt.io/full/v1/all_instruments)、[Bitget](https://api.bitget.com/api/v3/market/instruments?category=SPOT)、[GMX](https://arbitrum-api.gmxinfra.io/markets/info)。
