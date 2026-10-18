# market-kit 每日巡检：2026-10-18

覆盖今日线上 discovery 19,235 行、registry overrides 与官方补充目录的并集，共 19,800 个唯一市场、20 个平台。每条市场都有逐条 disposition、来源和身份边界检查，见各平台 JSONL。结构检查不等于独立核实每个产品，不是交易授权。相比昨日有 104 个新增覆盖 key，不直接称为新上市。

## 线上修复与部署验证

- 确认服务已部署 b924e9a（2026-10-17 构建），reviewed=1702。Binance ANTHROPIC 两种写法均恢复 resolved，Entropy io:ANTH 的共享 Anthropic 底层、独立产品与生命周期边界保留。用户已经确认其仓位为 io:ANTH；本次未操作 Portfolio-Lens 或声称页面实测。
- 完成此前受旧版本阻挡的 xyz:XYZ100 原生 canonical 修正为 XYZ100/USDC；股指 ID、乘数1、ambiguous 比较保持。
- gTrade 官方 forex / forex-minor / forex-exotic 共38个市场：此前20个归crypto、11个归fiat_stable、7个unknown。现按精确市场设为 FX 产品 ID、fx_cfd、multi_collateral_cfd，并禁止自动比较。当前接口无FX资产类别枚举，故 class 保留 unknown，不能用crypto或稳定币库存冒充。
- Extended 官方 RWA/FX 的 EUR-USD 和 USDJPY-USD 同样误归crypto，已改为独立 FX 产品并禁止比较。USDJPY 是参考资产，USD 是该目录的抵押/报价字段；未直接将它改写为跨平台 USD/JPY 或推断倍率。
- Bitget PREOPAIUSDT、PRESPCXUSDT 漏分类已修为rwa_stock。IPO Prime token 保留 native PRE 名称、独立产品 ID、ambiguous 比较，不合并到普通股票、Reality rToken 或永续。官方公告确认 preSPAX 曾按1:5拆分并改名preSPCX，未将新旧单位直接设为等价。
- 今日42项新增修复，另完成1项XYZ100旧阻塞。全部108条（含昨日修复）在最后一次写入后启动并成功完成的自动同步后回读，无验证差异。同步开始 2026-10-18T01:08:27.565352068Z，完成 2026-10-18T01:08:29.141124621Z；manualOverrideCount=108。已回读相关 assets；全局同名 alias 未批量改类。

## 完整目录与覆盖边界

- 今日 gTrade 官方 endpoint 恢复，493对、12分组全部核对，含62股票、10指数、8商品、38外汇，以及各crypto分组。此前TLS缺口已关闭。官方目录不是逐条可交易状态证明，保留unknown状态。
- Extended 404条perpetual、GRVT 184条active perpetual、GMX 126条可解析池证据（含地址和chain），均合入当日检查。GMX同base/quote不同池不自动合并。
- Bitget 官方2915个当前股票类市场全部核对，修复两个IPO Prime后无剩余类别差异。PAXG/XAUT保留商品敞口分类：官方CRYPTO属于设施类别，不据此覆盖实际商品底层。
- 全部80个线上解析批次（每批250、最多2并发）成功，无请求覆盖缺口。样本反馈源配置仍含api.example.com占位地址，不能声称覆盖真实消费者反馈。
- 60条历史RWA类别/crypto-ID线索仍保留待核查；抽查的旧Bitget合约未在当前官方目录出现，不因缺失就删除或视为当前可交易。295条crypto/unknown-ID线索大多已受ambiguous限制，不能当作确定同资产或错误计数。

## 代码与验证

- 官方外汇元数据优先于泛化crypto；不产生全局货币token alias，不从其他市场借用crypto/稳定币身份；官方非USD报价外汇也进入生成流程。FX比较保守禁止，未新增枚举以免破坏下游兼容。
- multi_collateral_cfd 没有确定结算资产时，解析器不再把quote当settlement。**此守卫仅在本地新代码中，当前线上 b924e9a 仍会返回报价币作为FX settlement。** 已写入的独立身份和禁止比较持续生效，不能声称结算字段完全修好；需部署本次提交，部署后38个gTrade FX的settlementAsset应为空。无需重复写规则或暂停同步。
- Bitget stock类别不再依赖isReality=yes；只有Reality rToken才去掉r前缀，IPO Prime保留原始产品名。
- Go全量race、vet、Linux amd64构建通过；20442行完整输入跨两轮同步保留1744条reviewed规则。相同输入与b924e9a生成器对比，仅新增34条FX、修改6条FX，无其他市场变化、删除或新增碰撞。
- 本机重放样本sync wall≈0.840s、CPU≈0.960s、reader p99≈3.63μs；不代表生产性能基准。没有修改ticker路径、同步频率、队列或丢弃政策；无额外实时网络请求。前端文件未变，本次未重新跑前端构建。
- bootstrap仍可能隐藏单一collector失败，属于保留的可观测性改进项；今日所有独立官方补查均成功。未擅自部署、重启或推送。

## 证据

fetch-manifest.json、inventory.json、registry.json、audit.json保留当日快照与初始自动诊断；各平台.jsonl保留最终逐市场结果；fx/extended-fx/ipo-repair-plan.json及对应online-write.json保留写前后摘要；final-online-verification.json保留108条跨同步回读及assets结果；generated-differential.json保留生成器差异。无凭据保存。

官方资料：[gTrade目录](https://backend-arbitrum.gains.trade/trading-variables)、[Extended目录](https://api.starknet.extended.exchange/api/v1/info/markets)、[GRVT目录](https://market-data.grvt.io/full/v1/all_instruments)、[Bitget目录](https://api.bitget.com/api/v3/market/instruments?category=SPOT)、[IPO Prime](https://www.bitget.com/asia/spotlight/ipo-prime)、[产品边界](https://www.bitget.com/asia/support/articles/12560603887172)、[preSPAX拆分公告](https://www.bitget.com/en-CA/support/articles/12560603884442)。
