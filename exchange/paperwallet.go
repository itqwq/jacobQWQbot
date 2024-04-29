package exchange

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/common"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// assetInfo 结构体表示单个资产的信息。
type assetInfo struct {
	Free float64 // 可用数量，指未锁定且可以自由交易的资产部分。
	Lock float64 // 锁定数量，指因交易（如挂单）而暂时不可用的资产部分。
}

// AssetValue 结构体用于记录某一时刻资产的价值。
type AssetValue struct {
	Time  time.Time // 记录价值的时间点。
	Value float64   // 该时间点资产的价值。
}

// PaperWallet 结构体模拟一个虚拟的钱包，用于无风险环境下的交易策略测试。
type PaperWallet struct {
	sync.Mutex                            // 互斥锁，用于确保多线程操作的数据一致性。
	ctx           context.Context         // 上下文，用于控制和管理子goroutine的生命周期。
	baseCoin      string                  // 基础货币，钱包的主要计价货币。 如果你将baseCoin设置为BTC，然后进行交易（比如卖出ETH），如果这笔交易亏损了，以BTC为单位计算的话，你的账户中显示的BTC数量确实会减少。
	counter       int64                   // 用于生成唯一的订单ID等。
	takerFee      float64                 // Taker手续费率。立即下单，不管卖出买入，产生的费用是takerFee
	makerFee      float64                 // Maker手续费率。下单的时候等待匹配(限价单)等待交易的过程产生的费用是makerFee
	initialValue  float64                 // 钱包初始价值。initialValue 表示的是钱包在模拟交易开始时的初始价值，它并不一定是0。这个值是在创建或初始化PaperWallet时由用户设定的
	feeder        service.Feeder          // 数据源，提供市场数据。
	orders        []model.Order           // 模拟的订单列表。
	assets        map[string]*assetInfo   // 持有的资产信息，键为资产标识符。
	avgShortPrice map[string]float64      //  做空的平均价值比如说当btc1800的时候我做空10 倍 1900的时候做空10倍，平均空头就是 (1800+1900)/2 =1850
	avgLongPrice  map[string]float64      // 你首先以10,000美元的价格买入了1 BTC，然后价格稍微下跌到9,500美元时，你又买入了1 BTC。此时，你的平均多头价格是：平均多头价格=10000 +9500 除以2 =9750美元btc
	volume        map[string]float64      // 记录每个货币对的交易量。
	lastCandle    map[string]model.Candle // 每个货币对的最后一个蜡烛图数据。最后等于最新数据
	fistCandle    map[string]model.Candle // 每个货币对的第一个蜡烛图数据，计算的是最初开盘的数据
	assetValues   map[string][]AssetValue // assetValues存储的正是每个交易货币在不同时间点上的价格信息。
	equityValues  []AssetValue            // 存储的信息包括了钱包或投资组合总价值随时间的任何变化，无论是盈利（赚）还是亏损（亏）
}

// AssetsInfo 方法接收一个货币对字符串（如"BTC/USD"）作为参数，并返回该货币对的相关资产信息。
func (p *PaperWallet) AssetsInfo(pair string) model.AssetInfo {
	// 使用 SplitAssetQuote 函数分解给定的货币对字符串为基础资产和计价资产。
	// 例如，对于"BTC/USD"，asset为"BTC"，quote为"USD"。
	asset, quote := SplitAssetQuote(pair)

	// 返回一个model.AssetInfo结构体实例，填充了货币对的详细信息：
	return model.AssetInfo{
		BaseAsset:          asset,           // 基础资产的标识符（如"BTC"）。
		QuoteAsset:         quote,           // 计价资产的标识符（如"USD"）。
		MaxPrice:           math.MaxFloat64, // 设置最大价格，这里使用了Go语言的最大浮点数表示无上限。
		MaxQuantity:        math.MaxFloat64, // 设置最大数量，同样使用了Go的最大浮点数表示无上限。
		StepSize:           0.00000001,      // 步长，表示数量的最小变化单位。
		TickSize:           0.00000001,      // 最小价格变动单位，即市场价格的最小变化幅度。
		QuotePrecision:     8,               // 计价资产精度，表示计价资产价格的小数点后位数。
		BaseAssetPrecision: 8,               // 基础资产精度，表示基础资产数量的小数点后位数。
	}
}

// PaperWalletOption 是一个配置函数的类型，用于自定义 PaperWallet 实例的设置。
type PaperWalletOption func(*PaperWallet)

// WithPaperAsset 配置特定货币对的初始资产量。
func WithPaperAsset(pair string, amount float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		// 初始化货币对资产，设置可用数量，锁定量为0。
		wallet.assets[pair] = &assetInfo{Free: amount, Lock: 0}
	}
}

// WithPaperFee 配置钱包的手续费率，包括maker手续费和taker手续费。
func WithPaperFee(maker, taker float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		// 设置maker手续费率。
		wallet.makerFee = maker
		// 设置taker手续费率。
		wallet.takerFee = taker
	}
}

// WithDataFeed 配置钱包使用的数据源。
func WithDataFeed(feeder service.Feeder) PaperWalletOption {
	return func(wallet *PaperWallet) {
		// 设置钱包的数据源为提供的feeder。
		wallet.feeder = feeder
	}
}

// NewPaperWallet 创建并初始化一个新的 PaperWallet 实例。 参数: ctx: 上下文，用于管理和取消长时间运行的操作。 baseCoin: 钱包的基础货币标识符，用于计价和资产评估。 options: 一个或多个配置选项，允许灵活地初始化钱包的不同方面。
func NewPaperWallet(ctx context.Context, baseCoin string, options ...PaperWalletOption) *PaperWallet {
	// 使用提供的参数和默认值初始化 PaperWallet 结构体。
	wallet := PaperWallet{
		ctx:           ctx,                           // 设置上下文
		baseCoin:      baseCoin,                      // 设置基础货币
		orders:        make([]model.Order, 0),        // 初始化订单列表
		assets:        make(map[string]*assetInfo),   // 初始化资产信息映射
		fistCandle:    make(map[string]model.Candle), // 初始化每个货币对的第一个蜡烛图数据映射
		lastCandle:    make(map[string]model.Candle), // 初始化每个货币对的最后一个蜡烛图数据映射
		avgShortPrice: make(map[string]float64),      // 初始化平均空头价格映射
		avgLongPrice:  make(map[string]float64),      // 初始化平均多头价格映射
		volume:        make(map[string]float64),      // 初始化交易量映射
		assetValues:   make(map[string][]AssetValue), // 初始化资产价值记录
		equityValues:  make([]AssetValue, 0),         // 初始化总权益价值记录
	}

	// 应用所有配置选项到钱包实例。
	for _, option := range options {
		option(&wallet) // 每个 option 是一个函数，修改 wallet 的配置
	}

	// 设置钱包的初始价值为基础货币的可用余额。
	wallet.initialValue = wallet.assets[wallet.baseCoin].Free

	// 记录钱包的使用和初始资产配置信息。
	log.Info("[SETUP] Using paper wallet") //正在使用纸钱包
	log.Infof("[SETUP] Initial Portfolio = %f %s", wallet.initialValue, wallet.baseCoin)

	// 返回初始化好的钱包实例指针。
	return &wallet
}

// 每次调用该方法，计数器 p.counter 会递增，
// 确保返回一个唯一的整数值作为订单 ID。
func (p *PaperWallet) ID() int64 {
	p.counter++
	return p.counter
}

// Pairs 返回模拟钱包中所有资产对的列表。
func (p *PaperWallet) Pairs() []string {
	pairs := make([]string, 0)   // 初始化一个空切片来存储货币对。
	for pair := range p.assets { // 遍历assets映射的键。
		pairs = append(pairs, pair) // 将货币对添加到切片中。
	}
	return pairs // 返回包含所有货币对的切片。
}

// LastQuote 返回指定货币对的最新报价。
func (p *PaperWallet) LastQuote(ctx context.Context, pair string) (float64, error) {
	return p.feeder.LastQuote(ctx, pair) // 从数据源获取最新报价。
}

// AssetValues 返回指定货币对的资产价值历史记录。
func (p *PaperWallet) AssetValues(pair string) []AssetValue {
	return p.assetValues[pair] // 返回指定货币对的价值记录。
}

// EquityValues 这个方法记录钱包的总余额 包括现金+货币
func (p *PaperWallet) EquityValues() []AssetValue {
	return p.equityValues // 返回整体资产的价值记录。
}

// MaxDrawdown方法的确是用来评估在一段时间内，模拟交易钱包中的资产价值从最高点下跌到最低点的最大幅度。
// 返回三个参数,最大回撤百分比,最大回撤开始的时间点 ,最大回撤结束的时间点
func (p *PaperWallet) MaxDrawdown() (float64, time.Time, time.Time) {
	// 钱包资产价值记录长度是否小于1，是因为没有进行任何交易，或者是模拟钱包刚刚初始化而还没有来得及记录资产价值的变化。
	if len(p.equityValues) < 1 {
		return 0, time.Time{}, time.Time{} // 如果没有资产价值记录，返回0和空时间。
	}

	//资产价值从一个峰值下降到随后的谷底的过程。每次从峰值到谷底的下降都可以视为一个局部回撤事件，全局回撤，多个局部回撤资产价值遭遇的最大下降幅度。
	localMin := math.MaxFloat64             //先把localMin设置成一个非常大的数字，这是为了在我们开始查找回撤之前有个参照点 ，这样，当我们遇到实际的下降（回撤）时，就能用这个下降值更新localMin。我们会一直更新localMin，直到找到观察期内最大的下降，也就是最大回撤。这样做确保我们最后得到的是整个期间最大的下降幅度。
	localMinBase := p.equityValues[0].Value // 局部回撤的基准价值（起始价值）基准价值是资产价值开始下降前的价值，即峰值。第一个元素获取起始值是因为这个切片按时间顺序记录了模拟交易钱包的资产价值变化。第一个元素代表了记录开始的点
	localMinStart := p.equityValues[0].Time // 局部回撤的起始时间。
	localMinEnd := p.equityValues[0].Time   // 局部回撤的结束时间。
	//全局回撤确实是指在观察期间内所有局部回撤中，从峰值下降到最低点的最大幅度。
	globalMin := localMin           // 全局最大回撤值初始化为最大的一个数。
	globalMinBase := localMinBase   // 全局回撤的基准价值。基准价值是资产价值开始下降前的价值，即峰值
	globalMinStart := localMinStart // 全局最大回撤的起始时间。
	globalMinEnd := localMinEnd     // 全局最大回撤的结束时间。

	// 遍历所有资产价值记录，计算最大回撤。
	for i := 1; i < len(p.equityValues); i++ {
		diff := p.equityValues[i].Value - p.equityValues[i-1].Value // diff表示当前记录与前一个记录之间的资产价值差异。正值表示上升 ，负值表示回撤。

		// 如果我们记录的局部最小回撤值还是一个很大的正数，这通常意味着我们还没开始真正追踪任何回撤  ，大于0 就没有记录回撤 因为没有更新
		if localMin > 0 {
			//把局部回撤值赋值给基准值
			localMin = diff
			localMinBase = p.equityValues[i-1].Value //就是我们用前面的那个点的价值作为起点，来看资产价值是怎么下降的，这个起点就是我们说的“基准价值”。
			localMinStart = p.equityValues[i-1].Time // 局部回撤的起始时间设置为前面的那个点的起始时间，因为，从上一个点开始上升，或回撤
			localMinEnd = p.equityValues[i].Time     //局部回撤的结束时间，为当前的结束时间
		} else {
			localMin += diff //localMin不大于0 ，是负数，表示已经在记录回撤，则继续累加diff，无论它是表示继续下降还是开始回升。重点在于追踪从峰值开始的整个下降过程，直至找到最大回撤。
			localMinEnd = p.equityValues[i].Time
		}

		// 如当前的局部回撤是-100 ，全局回撤是-50 所以说当前回撤比之前记录的全局回撤都要大，所以要更新
		if localMin < globalMin {
			globalMin = localMin           // 更新回撤的最大值给全局回撤
			globalMinBase = localMinBase   //更新基础回撤的起始点给全局回撤
			globalMinStart = localMinStart //更新全局回撤的开始时间
			globalMinEnd = localMinEnd     // 更新全局回撤的结束时间
		}
	}

	// 如果globalMin是-200（表示资产价值下降了200单位），而globalMinBase是1000（表示回撤前的峰值是1000单位），那么globalMin / globalMinBase就是-0.2。这意味着资产从峰值下降了20%（-0.2乘以100）到达最低点。返回全局最大回撤百分比及其起始和结束时间。
	return globalMin / globalMinBase, globalMinStart, globalMinEnd
}

// 这个Summary函数的目的就是为了汇总并解释模拟钱包的总体信息。它会给你展示整个模拟期间内钱包的表现，包括资产总值、市场的整体变化、盈亏情况、最大回撤（风险度量）以及交易活动的情况
func (p *PaperWallet) Summary() {
	// 初始化用于计算的变量：总资产价值、市场变化比率和交易量。
	var (
		total        float64 // 总资产价值。
		marketChange float64 // 市场变化比率，如果一个投资组合在某段时间开始时的价值是1000单位，而结束时的价值是1100单位，那么市场变化比率为:( 1100-1000 ) /1000 = 0.1 或10%  , 这就是市场变化率
		volume       float64 // 交易量。
	)

	fmt.Println("----- FINAL WALLET -----") // 打印最终钱包信息的标题。

	// 遍历每个货币对的最后一个蜡烛图（K线）数据。
	for pair := range p.lastCandle {
		asset, quote := SplitAssetQuote(pair) // 从货币对中分解出资产和计价货币。
		assetInfo, ok := p.assets[asset]      // 获取资产信息。
		if !ok {
			continue // 如果没有资产信息，跳过当前迭代。
		}

		// 计算资产的总量（可用量加锁定量）和价值。
		quantity := assetInfo.Free + assetInfo.Lock
		value := quantity * p.lastCandle[pair].Close // 资产的价值基于最后一根K线的收盘价。计算收盘价意味着计算我的总价值

		// 如果资产总量为负，说明是空头头寸，需要特殊处理。
		if quantity < 0 {
			//totalShort 的值为负，因此意味着平仓后是盈利的。 值为正，那么意味着平仓后亏了钱。
			//假设投资者做了一个空头头寸，借入了 10 个比特币（BTC），平均买入价格为 50000 美元/BTC，做空的时候比特币价格为 50000 美元/BTC。后来比特币价格下跌到 45000 美元/BTC，投资者决定平仓 totalShort := 5000 * -10 - 4500 * -10 = 500 赚取  ,2.0  表示额外的费用
			totalShort := 2.0*p.avgShortPrice[pair]*quantity - p.lastCandle[pair].Close*quantity
			value = math.Abs(totalShort)
		}

		// 更新总资产价值和市场变化比率。
		total += value
		marketChange += (p.lastCandle[pair].Close - p.fistCandle[pair].Close) / p.fistCandle[pair].Close
		fmt.Printf("%.4f %s = %.4f %s\n", quantity, asset, total, quote) // 打印每个资产的数量和价值。
	}

	// 计算市场平均变化比率。
	avgMarketChange := marketChange / float64(len(p.lastCandle))
	// 计算基础货币的最终价值。
	baseCoinValue := p.assets[p.baseCoin].Free + p.assets[p.baseCoin].Lock
	// 计算总盈利。盈利金额 = 整个投资组合的总价值 - 原始资产价值
	profit := total + baseCoinValue - p.initialValue

	// 打印总结信息。
	fmt.Printf("%.4f %s\n", baseCoinValue, p.baseCoin)
	fmt.Println()
	maxDrawDown, _, _ := p.MaxDrawdown() // 计算最大回撤。
	fmt.Println("----- RETURNS -----")   // 打印回报信息的标题。
	fmt.Printf("START PORTFOLIO     = %.2f %s\n", p.initialValue, p.baseCoin)
	fmt.Printf("FINAL PORTFOLIO     = %.2f %s\n", total+baseCoinValue, p.baseCoin)
	fmt.Printf("GROSS PROFIT        =  %f %s (%.2f%%)\n", profit, p.baseCoin, profit/p.initialValue*100)
	fmt.Printf("MARKET CHANGE (B&H) =  %.2f%%\n", avgMarketChange*100)

	fmt.Println()
	fmt.Println("------ RISK -------") // 打印风险信息的标题。
	fmt.Printf("MAX DRAWDOWN = %.2f %%\n", maxDrawDown*100)

	fmt.Println()
	fmt.Println("------ VOLUME -----") // 打印交易量信息的标题。
	// 计算总交易量。
	for pair, vol := range p.volume {
		volume += vol
		fmt.Printf("%s         = %.2f %s\n", pair, vol, p.baseCoin)
	}
	fmt.Printf("TOTAL           = %.2f %s\n", volume, p.baseCoin)
	fmt.Println("-------------------") // 结束标记。
}

// validateFunds 方法是为了在一个模拟钱包（PaperWallet）中验证是否有足够的资金来执行一个给定的交易，并在验证通过后更新钱包的资产状态。
// amount float64：这是用户想要买入或卖出的基础资产的数量，value float64：这个参数表示基础资产的当前市场价值 ，fill bool：这是一个布尔值，指示交易是否应该立即执行（即是否为市价交易）
func (p *PaperWallet) validateFunds(side model.SideType, pair string, amount, value float64, fill bool) error {
	// 从货币对中分解出资产和计价货币。
	asset, quote := SplitAssetQuote(pair)

	// 如果基础资产btc资产不在钱包中，则初始化资产信息。
	if _, ok := p.assets[asset]; !ok {
		p.assets[asset] = &assetInfo{}
	}

	// 如果计价货币usdt不在钱包中，则初始化资产信息。
	if _, ok := p.assets[quote]; !ok {
		p.assets[quote] = &assetInfo{}
	}

	// 获取计价货币的可用资金。
	funds := p.assets[quote].Free

	// 如果是卖出操作。
	if side == model.SideTypeSell {
		// 如果资产可用数量大于0，则将其加入到可用资金中。
		if p.assets[asset].Free > 0 {
			//假设基础资产btc 2个，每个1000元，计价资产原funds： USDT  5000 ，总价格等于funds =5000+ 2*1000 = 7000
			// 总资产 =  原计价资产 + 可用货币数量 * 单个货币价格
			funds += p.assets[asset].Free * value
		}

		// 这段代码的意图是检查钱包中的计价货币（在这个例子中是USDT）是否足够支付amount*value：想要购买的基础资产数量的总金额
		if funds < amount*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		//就是用先保证可用数量不为0，然后再用可用数量与想要交易的数量比较，取最小值，意思是拿到实际可以交易的值
		lockedAsset := math.Min(math.Max(p.assets[asset].Free, 0), amount)

		//是需要锁定的计价货币数量 = (是用户想要交易的基础资产数量-是实际可以交易的基础资产数量)*基础资产的市场价值
		//你想要卖出5 BTC，但只有3 BTC可用 差额是5-3 = 2btc ，表示如果有足够的btc 那么差额价值值20000
		//lockedQuote是差额乘以单价的计算结果，用于表示为了满足交易意向（买入或理论上的卖出）所需的额外计价货币金额。
		lockedQuote := (amount - lockedAsset) * value

		//更新交易后的基础货币可用数量 更新交易后基础资产可用数量 =  基础资产可用数量 -  实际可以交易的数量
		//就是看lockedAsset 用户想卖的是否比账户的多是吧 如果比账户的多就为0，比账户的少，就为正数
		p.assets[asset].Free -= lockedAsset
		//意思就是更新报价的总余额  用账户的报价余额- 额外需要的货币资金余额 =更新后的账户报价余额
		//额外金额本来就是 我账户的基础资产货币不足以交易我足够量的货如BTC，所以我再用账户的报价资产买BTC拿去市场上卖
		p.assets[quote].Free -= lockedQuote

		// 如果是立即成交，则更新平均价格并调整资产数量。
		if fill {
			// 更新平均价格
			p.updateAveragePrice(side, pair, amount, value)

			if lockedQuote > 0 { ////lockedQuote > 0 意思是我需要额外的钱买入这个如btc 因为实际基础数量不够，这通常在做空操作中发生，您借入资产然后卖出。
				p.assets[asset].Free -= amount // 实际存在的基础数量 - 想要卖的数量 = 账户还存在的基础货币数量（这是做空，已经介入足够对的基础数量去减掉想交易的数量了）
			} else { // 没有额外的计价货币需求，这意味着卖出操作的实际基础货币数量小于或等于你拥有的数量。
				p.assets[quote].Free += amount * value // 当lockedQuote <0  的时候表示 可以卖出实际的数量了，因为基础货币数量充足， 更新账户的报价资产数量 = 原来的报价资产数量 + 卖出的报价数量
			}
		} else {
			// 如果不是立即成交，当交易正在进行中的时候，把实际可交易的数量放到锁定数量中，第一可以保护交易过程中的安全，如果这是一个卖单，lockedAsset代表卖出的资产数量；这意味着这部分资产暂时不能用于其他交易，直到这个订单完成或被取消。
			p.assets[asset].Lock += lockedAsset
			//在做空的情况下把需要额外的需要的资金锁在账户中，确保有足够的资金做担保
			//尽管投资者可能有足够的资金作为保证金来“支持”他们的做空操作，但借入资产并卖出的核心目的是利用杠杆来赚取基于特定市场预期的利润。
			p.assets[quote].Lock += lockedQuote
		}

		log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	} else { // 如果是买入操作。 这里相当于 买入归还给交易所，平仓
		// 如果存在空头头寸，则计算清算空头头寸后的可用资金。
		var liquidShortValue float64
		//因为我觉得btc会下降所以我从平台借1个btc 然后账户变成-1，然后呢从更低得价格买入还给平台，从中赚取利润，所以p.assets[asset].Free < 0 是做空
		if p.assets[asset].Free < 0 {
			v := math.Abs(p.assets[asset].Free) //这个方法确实是将任何输入的数字都转换成相应的正数（或整数）。
			// BTC平均做空成本价10000 然后乘以2 等于20000比如做空下降到8000 我也可以盈利 ， 上涨到12000 本来是亏损的，显示我还是盈利 极端情况分析， 意思就是说平均做空成本2倍还能盈利，能够从一个更广泛的视角评估潜在的风险和盈亏情况
			//如果 liquidShortValue 为正，意味着在这种极端假设下，做空操作理论上仍能盈利；如果为负，说明做空操作将会亏损。
			liquidShortValue = 2*v*p.avgShortPrice[pair] - v*value
			//liquidShortValue 是正的 说明 盈利了 负的说明亏损了 ，然后再更新账户的报价资产funds
			funds += liquidShortValue
		}

		// 初始化本来想要买的数
		amountToBuy := amount
		//表示做空借入的数
		if p.assets[asset].Free < 0 {
			// 计算实际需要购买的数量， 本来想要买三个，账户还有一个做空，这样需要买四个 因为还有一个需要平仓之前借入的一个
			amountToBuy = amount + p.assets[asset].Free
		}

		// 如果可用资金不足以执行交易，则返回错误。
		if funds < amountToBuy*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		// 计算需要锁定的资产和计价货币数量。
		lockedAsset := math.Min(-math.Min(p.assets[asset].Free, 0), amount) //1、想买入的数量不能低于做空的数量，因为向平台借了要还，2、想买的数量达不到原本想买的总量举例子，本来想买入3个  仓位做空2个，还了还剩1个
		lockedQuote := (amount-lockedAsset)*value - liquidShortValue        //如计划买入5个，账户需要平仓空头3个，市场价格每个100， 空头盈利50，(5-3)*100-50=150 所以最后需要锁定150，需要平仓的是之前已经借了3个买入并立即将其卖出，获得资金，然后更低的价格买了还了，盈利了50，然后两个是需要买的,价格100, 200-50,  所以最后需要150预留资金锁定在账户买那两个就可以了

		// 更新资产信息：增加资产数量，减少计价货币数量。
		//lockedAsset是负数，将当时做空借入的数还了 ，然后更新账户中的基础资产数量
		p.assets[asset].Free += lockedAsset
		//表示更新账户的报价资产  = 原来的账户报价资产- 额外预留要买的基础资产
		p.assets[quote].Free -= lockedQuote

		// 如果是立即成交，则更新平均价格并调整资产数量。
		if fill {
			p.updateAveragePrice(side, pair, amount, value)
			//我明白了fill = true 已经立即完成交易了 所以说想要买的amount  数量已经完成了 ，所以减去做空平仓lockedAsset锁定的数量，剩下的放到基础资产仓库中
			p.assets[asset].Free += amount - lockedAsset
		} else {
			// 当挂单没有立即成交时，把需要做空的那一部分，锁在账户中，这样就确保能平仓，把卖单也锁住，没有取消交易之前，确保到这个价格，用户的挂单能正常交易
			p.assets[asset].Lock += lockedAsset
			//把额外需要购买额基础资产报价资产锁在账户里面，这样确保有一份钱能够购买这部分额外买的基础报价资产 这预防了因资金被其他用途消耗而导致的购买失败
			p.assets[quote].Lock += lockedQuote
		}
		log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	}

	return nil
}

// updateAveragePrice 根据交易的方向、货币对、数量和价值更新平均价格。
func (p *PaperWallet) updateAveragePrice(side model.SideType, pair string, amount, value float64) {
	//actualQty 表示交易前的实际持仓数量，初始化为0.0 意味着在交易之前没有任何持仓。随着交易的执行，actualQty 会根据交易的方向和数量进行更新，以反映交易执行后的最新持仓数量。
	actualQty := 0.0
	asset, quote := SplitAssetQuote(pair)

	// 基础资产(如btc)不为空，如果条件成立，意味着账户中存在该资产的仓位。更新交易后的数量。
	if p.assets[asset] != nil {
		//系统设计只关心自由数量，而不需要考虑锁定数量，那么可以不加上锁定数量。
		actualQty = p.assets[asset].Free
	}

	// 如果之前没有持仓
	if actualQty == 0 {
		// 如果是买入操作，意思确实是将传入的价格(value)更新为该货币对(pair)的平均多头持仓价格
		// 当你刚开始一个新的交易且之前没有任何多头或空头持仓时，传入的交易价格value就被用来设置作为这个新持仓的平均价格。
		if side == model.SideTypeBuy {
			p.avgLongPrice[pair] = value
		} else {
			// 如果交易的方向是卖出，那么就是更新平均空头的价格等于传入价格
			p.avgShortPrice[pair] = value
		}
		return
	}

	// 如果有持仓
	// 如果是实际做多加上买单
	//就是所买入一个自己看涨的股票，然后持有一段时间，涨到自已的预期了 卖出，从中赚取利润，这是做多
	if actualQty > 0 && side == model.SideTypeBuy {
		// 计算持仓价值 等于 平均多头价值 乘以 持仓数量
		positionValue := p.avgLongPrice[pair] * actualQty
		// 新平均价格 = (原持仓总价值 + 新增交易价值(打算交易数量×单个货币的价格)) / (原持仓数量 + 新增交易量)
		// 此公式用于更新做多的平均价格，不适用于更新做空的平均价格
		p.avgLongPrice[pair] = (positionValue + amount*value) / (actualQty + amount)
		return
	}

	// 这行代码检查是否存在多头仓位（即持仓量actualQty大于0）并且当前操作是卖出
	if actualQty > 0 && side == model.SideTypeSell {
		// 预期利润=(假设想要交易的数量×假设的价格)−(想要交易数量与仓位数量对比取最小值，即实际能交易的数量×平均购买价格)
		profitValue := amount*value - math.Min(amount, actualQty)*p.avgLongPrice[pair]

		// 预期利润百分比 = 预期利润 / (实际能够卖出的数量的成本)
		/// 注意：因为不管想交易多少数量（amount），都不能超过实际持仓的大小（actualQty），所以这里应当是基于实际能够卖出的数量计算的成本。
		// 这个比率可以帮助投资者评估在给定的交易计划下，相对于他们的投入成本，预期能够获得多少百分比的利润。
		// 高的预期利润百分比指示潜在的高盈利能力，但实际结果可能因市场条件和交易执行等因素而有所不同。
		percentage := profitValue / (amount * p.avgLongPrice[pair])
		// 输出利润信息4f：这表示格式化浮点数时保留四位小数。 percentage*100.0 表示原始的比率 0.1234 转换为常见的百分比形式为 12.34%
		log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0) // TODO: store profits

		// 想买的数量小于实际的数量，因为实际仓位数量在交易后还有仓位，里面的平均价格还是一样的所以不要额外操作，操作不会产生新的空头仓位，只是减少或完全平掉现有的多头仓位
		if amount <= actualQty { // not enough quantity to close the position
			return
		}
		// amount大于actualQty，即你试图卖出的数量超过了你实际持有的数量。不仅会平掉所有现有的多头持仓，还会创建新的空头持仓，因为你卖出了超过你持有的数量。这时，代码通过设置p.avgShortPrice[pair] = value来更新空头持仓的平均价格为当前卖出操作的价格（value）。空头平均价格等于想要卖出的价格意味着是第一次建立空头仓位。
		p.avgShortPrice[pair] = value
		return
	}

	// 如果是实际是空头寸加上卖单
	if actualQty < 0 && side == model.SideTypeSell {
		// 计算空头持仓价值 = 平均空头价值 *空头仓位的持仓数量
		positionValue := p.avgShortPrice[pair] * -actualQty
		// 平均空头价格=(实际原持仓价值+新增想要交易数量的总价值) /(实际持仓数量 + 想要交易的数量)
		p.avgShortPrice[pair] = (positionValue + amount*value) / (-actualQty + amount)

		return
	}

	// 如果是实际短头头寸加上买单
	// 做空：投资者借入100股该股票，并立即将其卖出，获得资金。股票价格下跌后，投资者以更低的价格买入100股股票，用于归还借入的股票 从中的差价就是利润
	if actualQty < 0 && side == model.SideTypeBuy {
		// 空头的假设利润 = (想要交易的数量与实际空头仓比拿最小值（即实际能够交易的数量））* 平均空头价格 - （想要交易的数量 * 当前价格）
		//假设以给定价格卖出空头仓位后可能获得的利润。这可以帮助你评估在当前情况下卖出空头仓位的潜在盈利或亏损。
		profitValue := math.Min(amount, -actualQty)*p.avgShortPrice[pair] - amount*value
		// 假设百分比 = 预期利润 / (实际能够卖出的数量的成本)
		//因为不管想交易多少数量（amount），都不能超过实际持仓的大小（actualQty），所以这里应当是基于实际能够卖出的数量计算的成本。
		// 好处，能够让投资者通过百分比知道预期收益的多少，百分比越高预期收入越高
		percentage := profitValue / (amount * p.avgShortPrice[pair])
		// 输出利润信息
		log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0) // TODO: store profits

		// 如果数量不足以关闭头寸，则返回
		if amount <= -actualQty { // not enough quantity to close the position
			return
		}

		// 更新长头头寸的平均价格
		p.avgLongPrice[pair] = value
	}
}

// OnCandle把蜡烛图的真实数据放到虚拟钱包中的过程，实际上是将市场的实时数据引入到模拟交易环境中，这样就可以根据这些真实的市场数据来指导模拟交易的决策和操作。这样可以让模拟交易环境能够模拟真实市场环境下的交易操作和资产管理。
func (p *PaperWallet) OnCandle(candle model.Candle) {
	// 锁定钱包，确保同时只有一个操作可以修改钱包的状态。
	p.Lock()
	// 函数执行完毕后解锁钱包。
	defer p.Unlock()

	// 更新最后一根蜡烛图的数据。
	p.lastCandle[candle.Pair] = candle
	// 如果不ok”，表示在p.fistCandle[candle.Pair]中还没有记录这个货币对的蜡烛图数据，也就是说，这是该货币对接收到的第一根蜡烛图。
	if _, ok := p.fistCandle[candle.Pair]; !ok {
		p.fistCandle[candle.Pair] = candle
	}

	// 遍历当前所有订单。
	for i, order := range p.orders {
		// 首先检查每个订单的货币对（如BTC/USDT）是否与最新蜡烛图的货币对相匹配。如果不匹配，跳过该订单，接下来检查订单是否处于新建状态，只有新建状态的订单才会根据最新的市场数据被考虑是否执行
		if order.Pair != candle.Pair || order.Status != model.OrderStatusTypeNew {
			continue
		}

		// 这段代码的作用是检查模拟交易钱包（PaperWallet）中是否已经记录了特定货币对（例如BTC/USDT）的交易量。如果没有（即之前没有对该货币对进行任何交易），则初始化该货币对的交易量为0 ，确保每个交易对都有一个交易量0
		if _, ok := p.volume[candle.Pair]; !ok {
			p.volume[candle.Pair] = 0
		}

		// 分解出基础资产和计价货币。
		asset, quote := SplitAssetQuote(order.Pair)
		// 如果是买入订单，且订单价格高于或等于当前蜡烛图的收盘价，则执行买入逻辑。说明在当前市场价格下，订单可以被执行（成交）。
		//如市场价是5000，订单价格如果低于$5,000，在真实市场中，这样的订单可能不会立即成交，因为它低于那个时间段市场买家愿意支付的最后价格。在模拟交易系统中，这种订单也可能不会被视为满足成交条件
		if order.Side == model.SideTypeBuy && order.Price >= candle.Close {
			// 如果账户中没有这种基础资产的记录，则初始化资产信息。
			if _, ok := p.assets[asset]; !ok {
				p.assets[asset] = &assetInfo{}
			}

			// 更新这个货币对的交易量 = 原货币对的交易量+ 订单的单价*订单的数量
			p.volume[candle.Pair] += order.Price * order.Quantity
			// 更新订单的最后更新时间，等于蜡烛图的更新时间
			p.orders[i].UpdatedAt = candle.Time
			// 更新订单状态为 完全成交：订单所有数量已成交
			p.orders[i].Status = model.OrderStatusTypeFilled

			// 更新账户中的资产数量和平均价格。
			p.updateAveragePrice(order.Side, order.Pair, order.Quantity, order.Price)
			// 增加基础资产的可用数量，就是这个订单已经买入了 所以更新原账户的基础资产数量 = 原数量+订单的数量
			p.assets[asset].Free = p.assets[asset].Free + order.Quantity
			// 减少计价货币的锁定数量，因为我挂单了买入了一些货币，所以目前账户的锁定报价资产= 锁定的报价资产-购买货币订单的那部分的报价资产
			p.assets[quote].Lock = p.assets[quote].Lock - order.Price*order.Quantity
		}

		// 如果是卖出订单的处理逻辑。
		if order.Side == model.SideTypeSell {
			var orderPrice float64
			// 根据订单类型和蜡烛图的高低点确定卖出价格。
			//订单类型为限价单、限价挂单、获利单或获利限价单之一 并且满足蜡烛图的最高价达到或超过了订单指定的价格 然后把订单的价格传给变量作为后续处理
			//设置这个订单 在当前最高价相等或低于一点执行订单类型操作，让交易者不错过盈利的同时，还能保证安全性
			if (order.Type == model.OrderTypeLimit || //限价单
				order.Type == model.OrderTypeLimitMaker || //限价挂单
				order.Type == model.OrderTypeTakeProfit || //获利单
				order.Type == model.OrderTypeTakeProfitLimit) && //获利限价单
				candle.High >= order.Price { //蜡烛图的最高价大于等于订单的价格
				orderPrice = order.Price // 在满足上述条件时，将订单的价格传给变量用于后续处理
			} else if (order.Type == model.OrderTypeStopLossLimit || //止损限价单
				order.Type == model.OrderTypeStopLoss) && //止损单
				candle.Low <= *order.Stop { //我设置了一个止损价为95，然后呢如果蜡烛图突然到了一个低点 ，低于95 ，然后就会执行止损价 这样保证在蜡烛图低于我设定的最低点冲破这个最低点，能够止损
				orderPrice = *order.Stop //如果上述两个条件都满足，那么订单的执行价格（orderPrice）将被设置为订单中指定的止损价格。
			} else {
				//如果不是止盈，止损单 就跳过
				continue
			}

			// 如果有订单组ID，取消同组的其他订单。
			if order.GroupID != nil {
				// 遍历所有订单groupOrder订单项 ，j为索引
				for j, groupOrder := range p.orders {
					//订单组将服务于同一策略的跨交易所订单整合，确保策略整体执行一致性。若组内某订单执行（如达成买卖条件），组内其他订单将取消，防止策略重复执行。这样，策略在不同平台的执行可自动同步，反映市场变化。
					//订单1如果有订单组 并且订单1的订单组等于订单2的订单组 并且不在一个交易所，执行下面操作
					if groupOrder.GroupID != nil && *groupOrder.GroupID == *order.GroupID &&
						groupOrder.ExchangeID != order.ExchangeID {
						p.orders[j].Status = model.OrderStatusTypeCanceled // 比如我在这个交易所的策略是买入订单，在其他交易所避免重复买入相同货币订单的策略，直接变成取消了，基于策略逻辑防止重复执行 通过ExchangeID表示订单所属的交易所标识。来确定哪个交易所的订单重复
						p.orders[j].UpdatedAt = candle.Time                //订单的最后更新时间等于蜡烛图的更新时间
						break                                              //结束退出
					}
				}
			}

			// 初始化计价货币的资产信息。
			if _, ok := p.assets[quote]; !ok {
				p.assets[quote] = &assetInfo{}
			}

			// 计算订单交易量。等于订单数量*订单的价格，看到底是止盈还是止损价
			orderVolume := order.Quantity * orderPrice

			// 更新交易量和订单状态。
			p.volume[candle.Pair] += orderVolume
			p.orders[i].UpdatedAt = candle.Time
			// 更新订单状态为已填充。
			p.orders[i].Status = model.OrderStatusTypeFilled

			// 更新账户中的资产数量和平均价格。
			p.updateAveragePrice(order.Side, order.Pair, order.Quantity, orderPrice)
			// 减少基础资产的锁定数量，因为卖出操作已经完成。所以挂单部分的报价锁定资产要减掉，更新账户
			p.assets[asset].Lock = p.assets[asset].Lock - order.Quantity
			// 增加计价货币的可用数量，因为卖出基础资产后收到了计价货币。
			p.assets[quote].Free = p.assets[quote].Free + order.Quantity*orderPrice
		}
	}

	// 如果蜡烛图完整，即当前周期结束，进行以下操作。
	if candle.Complete {
		var total float64 // 用于计算总资产价值。
		// 遍历所有资产，计算每种资产的价值。asset 是循环迭代过程中当前资产的键，而 info 则是该资产对应的值
		for asset, info := range p.assets {
			// 计算每种资产的总量（可用量 + 锁定量）。
			amount := info.Free + info.Lock
			// 构造该资产对应的交易对，以计算其价值。字符串中的所有字符转换为大写形式
			//如 p.baseCoin 表示基础货币是USDT， asset而当前资产是 BTC 就变成"BTCUSDT
			pair := strings.ToUpper(asset + p.baseCoin)
			// 如果资产数量为负（意味着做空），计算其清算价值。
			if amount < 0 {
				// 保证资产数量绝对值为正
				v := math.Abs(amount)
				//两倍的平均做空价格 - 最新拉组图的收盘价总价值 等于做空总价值
				liquid := 2*v*p.avgShortPrice[pair] - v*p.lastCandle[pair].Close
				//总价值= 原总价值(0) + 做空价值
				total += liquid
			} else {
				// 否则为做多，然后总资产等于，总数*蜡烛图*收盘价
				total += amount * p.lastCandle[pair].Close
			}

			// 记录每种资产的价值变动。
			p.assetValues[asset] = append(p.assetValues[asset], AssetValue{
				// 记录价值时间点
				Time: candle.Time,
				// 记录资产价值等于 总数*拉组图收盘价
				Value: amount * p.lastCandle[pair].Close,
			})
		}

		// 计算并记录总资产价值（包括基础货币）。
		baseCoinInfo := p.assets[p.baseCoin]
		p.equityValues = append(p.equityValues, AssetValue{
			// 记录时间点
			Time: candle.Time,
			// 记录基础货币(如USDT)总资产 =  做空或者做多得到的总资产+ 锁定资产+可用资产
			Value: total + baseCoinInfo.Lock + baseCoinInfo.Free,
		})
	}
}

// Account()用于获取模拟交易钱包（PaperWallet）的账户信息。
func (p *PaperWallet) Account() (model.Account, error) {
	// 初始化账户资产列表
	balances := make([]model.Balance, 0)
	// 遍历钱包中的资产信息，将每种资产的可用数量和锁定数量添加到资产列表中
	for pair, info := range p.assets {
		balances = append(balances, model.Balance{
			Asset: pair,      // 资产名称或标识符
			Free:  info.Free, // 可用资产数量
			Lock:  info.Lock, // 锁定资产数量
		})
	}

	// 构造并返回账户信息
	return model.Account{
		Balances: balances, // 账户资产列表
	}, nil
}

// Position 这个方法用于获取指定交易对在模拟交易钱包中的持仓信息。
func (p *PaperWallet) Position(pair string) (asset, quote float64, err error) {
	// 锁定钱包，确保同时只有一个操作可以修改钱包的状态。
	p.Lock()
	defer p.Unlock()

	// 分解交易对，获取基础资产和计价货币
	assetTick, quoteTick := SplitAssetQuote(pair)

	// 通过Account()获取交易对在模拟交易钱包当前账户信息
	acc, err := p.Account()
	if err != nil {
		return 0, 0, err
	}

	// 获取基础资产和计价货币的资产余额
	assetBalance, quoteBalance := acc.Balance(assetTick, quoteTick)

	// 返回基础资产和计价货币的总余额（包括可用数量和锁定数量）
	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// 这个方法用于创建一个OCO（One Cancels the Other）订单，即一个订单成交后会取消另一个订单。方法参数包括订单方向（买入或卖出）、交易对、数量、价格、止损价和止损限价。
func (p *PaperWallet) CreateOrderOCO(side model.SideType, pair string,
	size, price, stop, stopLimit float64) ([]model.Order, error) {
	// 锁定钱包，确保同时只有一个操作可以修改钱包的状态。
	p.Lock()
	defer p.Unlock()

	// 如果订单数量为0，则返回错误。
	if size == 0 {
		return nil, ErrInvalidQuantity
	}

	// 验证资金是否足够下单。
	err := p.validateFunds(side, pair, size, price, false)
	if err != nil {
		return nil, err
	}

	// 为两种订单生成相同的订单组ID。 不能使用同一个策略交易
	groupID := p.ID()

	// 生成限价挂单。
	limitMaker := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeLimitMaker,
		Status:     model.OrderStatusTypeNew,
		Price:      price,
		Quantity:   size,
		GroupID:    &groupID, //订单组
		RefPrice:   p.lastCandle[pair].Close,
	}

	// 生成止损单。
	stopOrder := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeStopLoss,
		Status:     model.OrderStatusTypeNew,
		Price:      stopLimit,
		Stop:       &stop,
		Quantity:   size,
		GroupID:    &groupID,
		RefPrice:   p.lastCandle[pair].Close, //止损单的参考价值的是蜡烛图最新更新的收盘价
	}

	// 将生成的订单添加到钱包的订单列表中。
	p.orders = append(p.orders, limitMaker, stopOrder)

	// 返回生成的两个订单。
	return []model.Order{limitMaker, stopOrder}, nil
}

// CreateOrderLimit 创建限价订单。side 表示订单的交易方向（买入或卖出）。pair 表示交易对。size 表示订单的数量。limit 表示订单的限价。返回创建的订单和可能出现的错误。
func (p *PaperWallet) CreateOrderLimit(side model.SideType, pair string,
	size float64, limit float64) (model.Order, error) {

	// 在进行操作前加锁以确保数据一致性，操作完成后解锁。
	p.Lock()
	defer p.Unlock()

	// 如果订单数量为零，则返回错误。
	if size == 0 {
		return model.Order{}, ErrInvalidQuantity
	}

	// 验证资金是否充足以进行交易。
	err := p.validateFunds(side, pair, size, limit, false)
	if err != nil {
		return model.Order{}, err
	}

	// 创建订单对象。
	order := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeLimit,
		Status:     model.OrderStatusTypeNew,
		Price:      limit,
		Quantity:   size,
	}

	// 将新订单添加到订单列表中。
	p.orders = append(p.orders, order)

	// 返回创建的订单和没有错误。
	return order, nil
}

// CreateOrderMarket 创建市价订单。
func (p *PaperWallet) CreateOrderMarket(side model.SideType, pair string, size float64) (model.Order, error) {
	// 在进行操作前加锁以确保数据一致性，操作完成后解锁。
	p.Lock()
	defer p.Unlock()

	// 调用 createOrderMarket 方法创建市价订单并返回结果。
	return p.createOrderMarket(side, pair, size)
}

// CreateOrderStop 创建止损限价订单。
func (p *PaperWallet) CreateOrderStop(pair string, size float64, limit float64) (model.Order, error) {
	// 在进行操作前加锁以确保数据一致性，操作完成后解锁。
	p.Lock()
	defer p.Unlock()

	// 如果订单数量为零，则返回错误。
	if size == 0 {
		return model.Order{}, ErrInvalidQuantity
	}

	// 验证资金是否足够下单。
	err := p.validateFunds(model.SideTypeSell, pair, size, limit, false)
	if err != nil {
		return model.Order{}, err
	}

	// 创建止损限价订单。
	order := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       model.SideTypeSell,
		Type:       model.OrderTypeStopLossLimit,
		Status:     model.OrderStatusTypeNew,
		Price:      limit,
		Stop:       &limit,
		Quantity:   size,
	}
	// 将订单添加到钱包中的订单列表中。
	p.orders = append(p.orders, order)
	return order, nil
}

// createOrderMarket 创建市价订单。
func (p *PaperWallet) createOrderMarket(side model.SideType, pair string, size float64) (model.Order, error) {
	// 如果订单数量为零，则返回错误。
	if size == 0 {
		return model.Order{}, ErrInvalidQuantity
	}

	// 验证资金是否足够下单。
	err := p.validateFunds(side, pair, size, p.lastCandle[pair].Close, true)
	if err != nil {
		return model.Order{}, err
	}

	// 如果交易对的成交量还没有记录，则初始化为零。
	if _, ok := p.volume[pair]; !ok {
		p.volume[pair] = 0
	}

	// 更新交易对的成交量。 = 原成交量+ 交易对最新蜡烛图的收盘价*交易对订单数量
	p.volume[pair] += p.lastCandle[pair].Close * size

	// 创建市价订单。
	order := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeMarket,
		Status:     model.OrderStatusTypeFilled,
		Price:      p.lastCandle[pair].Close,
		Quantity:   size,
	}

	// 将订单添加到钱包中的订单列表中。
	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderMarketQuote 创建一个以报价货币数量为基准的市价订单
// 在这个CreateOrderMarketQuote方法内部先把报价资产(如USDT)总数转化中基础资产(如btc)总数，然后调用createOrderMarket创建市场订单
func (p *PaperWallet) CreateOrderMarketQuote(side model.SideType, pair string, quoteQuantity float64) (model.Order, error) {
	// 锁定钱包以确保线程安全，函数结束时解锁。
	p.Lock()
	defer p.Unlock()

	// 获取交易对的资产信息。
	info := p.AssetsInfo(pair)

	// 计算基础货币的数量，使用普通货币的数量除以当前蜡烛图的收盘价，并根据交易对的最小交易量和基础货币精度进行取整。common.AmountToLotSize 规范货币的规则比如最小精度等
	//如1个USDT购买比特币（BTC），当前蜡烛图的BTC/USDT的收盘价为50 基础资产数量 = 1 USDT / 50 USDT/BTC = 0.02 BTC
	//quantity 是指购买基础资产的数量，经过 common.AmountToLotSize 函数处理后，确保符合所需的最小交易量要求和基础资产的精度要求。
	quantity := common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, quoteQuantity/p.lastCandle[pair].Close)

	// 调用内部函数创建市价订单。
	return p.createOrderMarket(side, pair, quantity)
}

// 这个方法用于取消指定的订单。
func (p *PaperWallet) Cancel(order model.Order) error {
	// 锁定钱包以确保线程安全，函数结束时解锁。
	p.Lock()
	defer p.Unlock()

	// 遍历钱包中的订单列表，寻找与传入订单相同交易所ID的订单，并将其状态设为已取消。
	for i, o := range p.orders {
		if o.ExchangeID == order.ExchangeID {
			p.orders[i].Status = model.OrderStatusTypeCanceled
		}
	}
	return nil
}

// 这个方法用于在钱包中查找特定ID的订单。它接收一个订单ID作为参数
func (p *PaperWallet) Order(_ string, id int64) (model.Order, error) {
	// 遍历钱包中的订单列表，查找与指定ID匹配的订单。
	for _, order := range p.orders {
		if order.ExchangeID == id { // 如果订单的ExchangeID与指定ID匹配，则返回该订单。
			return order, nil
		}
	}
	// 如果未找到匹配的订单，则返回一个空订单和相应的错误信息。
	return model.Order{}, errors.New("order not found")
}

// CandlesByPeriod 用于获取指定时间段内给定货币对的K线（蜡烛图）数据。
func (p *PaperWallet) CandlesByPeriod(ctx context.Context, pair, period string, start, end time.Time) ([]model.Candle, error) {
	// 通过PaperWallet的feeder属性直接调用CandlesByPeriod方法。
	// 这里的feeder很可能是对外部数据源（例如，一个交易所API）的接口。
	return p.feeder.CandlesByPeriod(ctx, pair, period, start, end)
}

// CandlesByLimit 用于获取指定时间周期内给定货币对的最新K线（蜡烛图）数据，数量上限由limit参数指定。
func (p *PaperWallet) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]model.Candle, error) {
	// 通过调用PaperWallet实例的feeder属性的CandlesByLimit方法来实现。
	// 这里的feeder属性可能是一个接口，负责与提供K线数据的外部数据源（如交易所API）进行交互。
	return p.feeder.CandlesByLimit(ctx, pair, period, limit)
}

// CandlesSubscription 创建一个实时订阅，用于获取指定货币对和时间框架的K线数据。
func (p *PaperWallet) CandlesSubscription(ctx context.Context, pair, timeframe string) (chan model.Candle, chan error) {
	// 通过调用PaperWallet实例的feeder属性的CandlesSubscription方法来实现订阅。
	// 这里的feeder属性可能是一个接口，负责与提供实时K线数据的外部数据源（如交易所API）进行交互。
	return p.feeder.CandlesSubscription(ctx, pair, timeframe)
}
