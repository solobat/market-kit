# 2026-11-04 北京时间巡检

覆盖线上 discovery inventory 与 registry 并集19,827条市场键、20个平台；补查原生 Aster/Lighter/Backpack 目录40条新键，总计19,867条有逐条 disposition。另存6条缺失 contractType 的 Aster 元数据，未冒充已解析市场。主输入21,578条证据，批量250、最多2并发。Extended416、gTrade493（全部类别）、GRVT184、GMX126、Gate、Bybit交割合约及全部 Hyperliquid dex 官方目录均有当日来源。resolved/无 flags 不代表独立身份和交易可比性核实成功。

在线修复6条（精确市场范围，compact=1）：Bitget MAUSDT 的 isRwa YES 仅证明RWA，移除仅凭 ticker 共享可比资格，保留股票产品分类及平台身份，ambiguous；Bybit USDBRLUSDT 官方 forex / US Dollar to Brazil Real / LinearPerpetual / USDT，移除股票误类，使用 unknown（无FX枚举）、fx_perpetual、平台范围prohibited；Variational EIDO 官方名 iShares MSCI Indonesia ETF，crypto改为rwa_stock，平台范围ambiguous，不推断份额换算；Aster SHROOMUSDT、PAIRUSDT、MAXUSDT 官方 SETTLING，精确市场prohibited，保留既有类别/底层但不认证，重新启用须官方活跃及身份单位复核。逐批写入前后与最终同步验证见 stock/inactive/eido-online-write.json、online-write.json、final-online-verification.json。不得从状态推断历史下架日期。

新增19个主市场键；Bybit AKAM/CEG/PFE已由原生股票规则保持平台范围ambiguous；Hyper io:GOOGL 保持prohibited。io:CRCL官方isDelisted标识消失，未据此新启用，当前本已eligible，待单位身份复核。Bitget RFFOG/RFGDL补充ISIN、RFGDL更名线索，不据此合并/推算兑换比例。COPPER旧待核实项保留。

线上采集19,249条，比昨日18,025恢复1,224条；Aster/Lighter/Lighter-Robinhood/Variational已重返清单。Lighter-Robinhood独立原生目录仍未核实；主Lighter目录不能代替。线上仍4efa310、schema10，失败源遥测尚未部署，cacheError为空不证明无部分源失败。

代码补充官方 Bybit forex 元数据优先于泛化股票标签，限定原生来源/LinearPerpetual/perpetual；复用已有FX隔离规则。生成版本升11，部署后重建旧动态层。回归覆盖双输入顺序和不可信来源/交割/现货反例。无新轮询、队列或丢弃/陈旧降级。

验证：提交视图（仅将用户既有aggregator两文件映射HEAD的只读编译overlay）完整Go测试、vet、构建、6个Python测试通过。工作区仍有昨日已知服务器分组测试差异3 vs 2，用户文件未改动。完整21,578行本地同步回放 wall690ms、CPU799ms、读p50 666ns/p99 3125ns、heap150.4MB；单次本地测量非生产CPU改善结论；不涉及JS事件循环。未调整刷新/丢弃策略。最终同步验证以JSON fresh=true及无errors为准。

所有原始抓取有当日时间/URL，逐市场结果在dispositions.jsonl；历史证据只做差异基线。未推送、未部署、未重启。
