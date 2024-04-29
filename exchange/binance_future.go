package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/jpillora/backoff"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// 保证金定义杠杆类型
type MarginType = futures.MarginType

// 定义杠杆类型常量
var (
	MarginTypeIsolated MarginType = "ISOLATED" // 独立杠杆你为某个特定的交易或货币对单独准备了一份钱。如果这个交易亏损，亏的只是这部分钱，不会影响到你账户中的其他资金。
	MarginTypeCrossed  MarginType = "CROSSED"  // 交叉全仓杠杆当你在某个交易中的资金快要亏完时，系统会自动使用你账户中的其他可用资金来补充保证金，以避免被强制平仓。

	ErrNoNeedChangeMarginType int64 = -4046 // 不需要改变杠杆类型的错误码1.以是当前杠杆的情况下，2.没有持仓影响
)

// PairOption 定义了交易对的配置选项
type PairOption struct {
	Pair       string             // 交易对
	Leverage   int                // 杠杆倍数
	MarginType futures.MarginType // 杠杆类型
}

// BinanceFuture 结构体定义了Binance期货交易的客户端
type BinanceFuture struct {
	ctx        context.Context            // 上下文可以用来控制和取消长时间运行的操作，如HTTP请求等。
	client     *futures.Client            // 这是与Binance Futures API进行通信的客户端实例，相当于一个桥梁用于执行实际的API调用，如下单、查询账户信息等。
	assetsInfo map[string]model.AssetInfo // 交易对的资产信息 键是交易对名称，值是交易的信息，如最小数量。
	HeikinAshi bool                       // 是否使用Heikin Ashi蜡烛图
	Testnet    bool                       // 是否使用测试网

	APIKey    string // API密钥（Key）可以被视为一个特殊的用户名或标识符，它唯一标识了API的调用者。
	APISecret string //  API密钥的秘密（Secret）部分可以被视为密码。

	MetadataFetchers []MetadataFetchers // 元数据获取器 这个方法或者工具被用来“抓取”或“获取”交易相关的额外信息，也就是元数据。元数据可以是任何有助于分析或决策的额外数据，比如交易对的历史表现、市场趋势、交易量分析等。
	PairOptions      []PairOption       // 交易对选项PairOptions是一个切片装填着多个交易对的杠杆信息 里面有交易对 杠杆倍数 ，杠杆类型
}

// BinanceFutureOption 定义了BinanceFuture构造函数的选项类型 这个类型接收一个指针的参数，就可以修改指针的所有的数据
type BinanceFutureOption func(*BinanceFuture)

// WithBinanceFuturesHeikinAshiCandle 启用Heikin Ashi蜡烛图
func WithBinanceFuturesHeikinAshiCandle() BinanceFutureOption {
	return func(b *BinanceFuture) {
		b.HeikinAshi = true
	}
}

// WithBinanceFutureCredentials 设置Binance Futures的凭证
func WithBinanceFutureCredentials(key, secret string) BinanceFutureOption {
	return func(b *BinanceFuture) {
		b.APIKey = key
		b.APISecret = secret
	}
}

// WithBinanceFutureLeverage 设置交易对的杠杆，leverage杠杆倍数，marginType杠杆类型
func WithBinanceFutureLeverage(pair string, leverage int, marginType MarginType) BinanceFutureOption {
	return func(b *BinanceFuture) {
		b.PairOptions = append(b.PairOptions, PairOption{
			Pair:       strings.ToUpper(pair), //ToUpper它接受一个字符串参数，并返回该字符串的全大写版本。
			Leverage:   leverage,
			MarginType: marginType,
		})
	}
}

// 这段代码实现了一个用于初始化并配置一个与Binance Futures API交互的交易机器人的功能。这个过程不仅仅是获取交易所的资产信息，而是涵盖了多个步骤，用于创建、配置，并启动一个为期货交易设计的交易机器人的实例。
// NewBinanceFuture 创建一个新的 BinanceFuture 实例。
// options ...BinanceFutureOption: 表示这个函数可以接受零个或多个BinanceFutureOption类型的参数。
func NewBinanceFuture(ctx context.Context, options ...BinanceFutureOption) (*BinanceFuture, error) {
	// 设置 WebsocketKeepalive 标志为 true，表示启用 Websocket 连接的保活功能交易机器人。以交易所webSocket协议允许这两者之间建立一个持久的、实时的通信通道。
	binance.WebsocketKeepalive = true

	// 上下文在这里像是一个多功能的遥控器，允许开发者在需要时对交易机器人进行精确的控制，无论是取消操作、设置超时，还是传递重要的控制信息。这个遥控器（即上下文）告诉BinanceFuture实例 1.何时停止等待 2.操作的超时时间 3.传递必要的信息 exchange一开始就是BinanceFuture的一个实例，它确实拥有BinanceFuture里定义的所有属性。不过，在这个初始时刻，我们只明确设置了它的ctx（上下文）属性。
	exchange := &BinanceFuture{ctx: ctx}

	// 每个option，都是一个函数BinanceFutureOption，通过传入exchange相当于BinanceFuture配置 ，所以可以通过这个函数 option   修改 exchange 里面所有的配置
	for _, option := range options {
		option(exchange)
	}

	// exchange.client 是机器人的一部分，使用提供的 API 密钥和秘钥创建一个 futures.Client 实例，用于后续的 API 调用。这段代码的作用就是利用交易所提供的API密钥和秘钥，让交易机器人建立与交易所的连接。
	exchange.client = futures.NewClient(exchange.APIKey, exchange.APISecret)

	// NewPingService定于币安API客户端库的，向币安服务器发送 Ping 请求，检查与服务器的连接是否正常Do(ctx)，Do(ctx)相当一个控制器  在记录连接的同时 还可以下达命令 如取消操作，设置超时、截至时间等
	err := exchange.client.NewPingService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("binance ping fail: %w", err)
	}

	// NewExchangeInfoService是定于币安API客户端库的，获取交易所的交易对信息和交易限制如交易对、价格精度、数量精度、最小交易量、最大交易量、最小和最大价格、杠杆信息等
	results, err := exchange.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, err
	}

	// 根据配置的交易对选项，设置每个交易对的杠杆和杠杆类型
	// 遍历exchange.PairOptions 得到里面的 交易对 杠杆倍数 ，杠杆类型
	for _, option := range exchange.PairOptions {
		// 为指定的交易对option.Pair设置杠杆大小option.Leverage Symbol(option.Pair)相当于交易对如symbol := "BTC/USDT"
		_, err = exchange.client.NewChangeLeverageService().Symbol(option.Pair).Leverage(option.Leverage).Do(ctx)
		if err != nil {
			return nil, err
		}
		// 设置交易对的杠杆类型比如说单独杠杆，还有交叉杠杆
		err = exchange.client.NewChangeMarginTypeService().Symbol(option.Pair).MarginType(option.MarginType).Do(ctx)
		if err != nil {
			// 把err变成*common.APIError 类型，如果返回false或者错误码为 ErrNoNeedChangeMarginType，则表示不需要修改杠杆类型，否则返回错误，*common.APIError 类型，可能包含了一些用于描述 API 错误的字段，比如错误码、错误信息等。
			if apiError, ok := err.(*common.APIError); !ok || apiError.Code != ErrNoNeedChangeMarginType {
				return nil, err
			}
		}
	}

	// 初始化一个映射，用于存储每个交易对的资产信息，包括基础资产、报价资产、精度和交易限制等
	exchange.assetsInfo = make(map[string]model.AssetInfo)
	// 遍历交易所里面的资产信息，然后拿到值，给我们自定义的资产信息
	for _, info := range results.Symbols {
		tradeLimits := model.AssetInfo{
			BaseAsset:          info.BaseAsset,
			QuoteAsset:         info.QuoteAsset,
			BaseAssetPrecision: info.BaseAssetPrecision,
			QuotePrecision:     info.QuotePrecision,
		}
		// 通过遍历 info.Filters，你可以获取到币安交易所中针对每个交易对设置的交易限制信息。每个交易对都可能有不同的限制，例如最小交易量、最大交易量、价格精度等。
		for _, filter := range info.Filters {
			//filterType 是用来表示交易限制的类型的字段。
			if typ, ok := filter["filterType"]; ok {
				// 如果当前的过滤器为SymbolFilterTypeLotSize，用于表示交易对的过滤器类型为交易量过滤器。这种过滤器用于限制交易对的最小交易数量、最大交易数量和交易数量的步长。则执行下面的操作
				if typ == string(binance.SymbolFilterTypeLotSize) {
					// 这行代码的目的是从交易限制过滤器中提取最小交易数量，并将其转换为浮点数，以便在代码中进行进一步处理和使用。
					tradeLimits.MinQuantity, _ = strconv.ParseFloat(filter["minQty"].(string), 64)
					tradeLimits.MaxQuantity, _ = strconv.ParseFloat(filter["maxQty"].(string), 64)
					tradeLimits.StepSize, _ = strconv.ParseFloat(filter["stepSize"].(string), 64)
				}
				// 检查当前过滤器的类型是否为价格过滤器类型。
				if typ == string(binance.SymbolFilterTypePriceFilter) {
					tradeLimits.MinPrice, _ = strconv.ParseFloat(filter["minPrice"].(string), 64)
					tradeLimits.MaxPrice, _ = strconv.ParseFloat(filter["maxPrice"].(string), 64)
					tradeLimits.TickSize, _ = strconv.ParseFloat(filter["tickSize"].(string), 64)
				}
			}
		}
		// info.Symbol是键，而 tradeLimits 是值。这行代码的意思是将交易对的资产信息存储到 exchange.assetsInfo 这个映射中值是包含基础资产、报价资产、精度和交易限制等信息的 tradeLimits 结构体。
		exchange.assetsInfo[info.Symbol] = tradeLimits
	}

	// 记录了一条日志，说明成功配置并且正在使用币安期货交易所。
	log.Info("[SETUP] Using Binance Futures exchange")

	// 返回配置好的 BinanceFuture 实例以及 nil 表示没有错误发生
	return exchange, nil
}

// LastQuote 返回指定交易对最近一次报价的收盘价。
// 它接受一个上下文和一个交易对作为参数，并返回最近一次报价的收盘价以及可能的错误。
func (b *BinanceFuture) LastQuote(ctx context.Context, pair string) (float64, error) {
	// 调用 b.CandlesByLimit方法获取指定交易对的最近一条K线数据，由于 limit 参数设置为 1，它只会返回最近的一条K线数据，也就是最后一根K线。
	candles, err := b.CandlesByLimit(ctx, pair, "1m", 1)
	// 检查是否发生错误或者获取到的K线数据长度是否小于1。
	// 如果发生错误或者K线数据长度小于1，返回0和错误。
	if err != nil || len(candles) < 1 {
		return 0, err
	}
	// 返回最近一条K线数据的收盘价作为最近一次报价的收盘价，并且返回nil表示没有错误。
	return candles[0].Close, nil
}

// 通过交易对然后得到资产信息pair 是一个键 通过这个键得到资产信息
func (b *BinanceFuture) AssetsInfo(pair string) model.AssetInfo {
	return b.assetsInfo[pair]
}

// 这个validate方法专门用于校验交易数量是否位于交易所规定特定交易对允许的最小和最大交易数量范围内。
func (b *BinanceFuture) validate(pair string, quantity float64) error {
	// 从 b.assetsInfo 中尝试获取指定交易对的资产信息。
	info, ok := b.assetsInfo[pair]
	// 如果获取失败（即交易对不存在于资产信息映射中），返回 ErrInvalidAsset 错误。
	if !ok {
		return ErrInvalidAsset
	}

	// 如果传入的交易数量超出了资产信息中定义的最大数量或小于最小数量，
	// 则构造并返回一个包含错误信息的 OrderError 类型错误。
	//把错误交易对赋值给OrderError，提供了出错的详细上下文，还便于调用方理解出错的原因和如何解决。
	if quantity > info.MaxQuantity || quantity < info.MinQuantity {
		return &OrderError{
			Err:      fmt.Errorf("%w: min: %f max: %f", ErrInvalidQuantity, info.MinQuantity, info.MaxQuantity), // 创建一个包含错误信息的 fmt.Errorf，说明了是无效的数量，同时指出了最小和最大允许的数量。
			Pair:     pair,                                                                                      // 交易对名称
			Quantity: quantity,                                                                                  // 提供的交易数量
		}
	}

	// 如果交易数量在允许的范围内，则不返回任何错误（返回 nil）。
	return nil
}

// 一个止盈单（盈利单）和一个止损单（亏损单）。这两个订单被同时下达，但一旦其中一个条件被触发并且订单执行，另一个订单会自动被取消。
// 使用panic("not implemented")和将所有参数替换为下划线 _ 明确地表明了这个功能还没有被开发或实现。
func (b *BinanceFuture) CreateOrderOCO(_ model.SideType, _ string,
	_, _, _, _ float64) ([]model.Order, error) {
	// 方法直接触发一个panic，说明这个功能没有被实现。
	panic("not implemented")
}

// CreateOrderStop创建一个止损订单。
// pair指定了交易对，比如"BTCUSDT"。
// quantity是想要交易的数量。
// limit是止损价格。
func (b *BinanceFuture) CreateOrderStop(pair string, quantity float64, limit float64) (model.Order, error) {
	// 首先，使用validate方法校验提供的交易对和数量是否有效。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果校验失败，则返回一个空的Order对象和错误信息。
		return model.Order{}, err
	}

	// 使用客户端的NewCreateOrderService方法准备创建新的订单。
	order, err := b.client.NewCreateOrderService().
		Symbol(pair).                               // 设置订单的交易对。
		Type(futures.OrderTypeStopMarket).          // 设置订单类型为止损市价单。
		TimeInForce(futures.TimeInForceTypeGTC).    //根据市场的订单类型（如限价订单、市价订单）直到被成交，或者被用户手动取消。
		Side(futures.SideTypeSell).                 // 设置订单方向为卖出。
		Quantity(b.formatQuantity(pair, quantity)). // 设置订单数量，使用formatQuantity方法格式化。
		Price(b.formatPrice(pair, limit)).          // 格式化价格的目的是确保价格符合交易所的要求，并且符合交易对的价格精度规则。需要确保价格的精度符合交易所的规定，否则拒绝执行
		Do(b.ctx)                                   // 调用 Do(b.ctx) 方法并传入上下文会触发客户端向交易所发送订单请求。
	if err != nil {
		// 如果创建订单请求失败，则返回空的Order对象和错误信息。
		return model.Order{}, err
	}

	// 解析订单价格和原始数量的值。
	price, _ := strconv.ParseFloat(order.Price, 64)
	quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)

	// 构造并返回一个Order对象，包含订单的详细信息。
	return model.Order{
		ExchangeID: order.OrderID,                                          // 订单在交易所的ID。
		CreatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单创建时间。将订单的最后更新时间转换为毫秒级别，然后符合国际标准的时间格式。
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单最后更新时间。
		Pair:       pair,                                                   // 交易对。
		Side:       model.SideType(order.Side),                             // 订单方向（买/卖）。
		Type:       model.OrderType(order.Type),                            // 订单类型。
		Status:     model.OrderStatusType(order.Status),                    // 订单状态。
		Price:      price,                                                  // 订单价格。
		Quantity:   quantity,                                               // 订单数量。
	}, nil
}

// formatPrice 就是保证价格变动能在交易所的指定的值内 比如上涨0.01 就必须保证变动是这个
func (b *BinanceFuture) formatPrice(pair string, value float64) string {
	if info, ok := b.assetsInfo[pair]; ok {
		value = common.AmountToLotSize(info.TickSize, info.QuotePrecision, value)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// formatQuantity方法用于格式化给定的交易量值，以确保它符合币安期货交易对的要求。
func (b *BinanceFuture) formatQuantity(pair string, value float64) string {
	// 尝试从b.assetsInfo中获取指定交易对的资产信息。
	if info, ok := b.assetsInfo[pair]; ok {
		// 如果成功找到交易对的信息common.AmountToLotSize方法的作用是确保你指定的交易量符合特定的规则。
		// info.StepSize表示每次交易量变化的最小单位。
		// info.BaseAssetPrecision表示基础资产数量的精度。
		// 这里将value传入的参数可能不符合规定数量的规则所以调用AmountToLotSize方法传入三个参数调整
		value = common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, value)
	}

	// 使用strconv.FormatFloat将调整后的float64类型的交易量值格式化为字符串。
	// 'f'表示使用小数点形式而非科学计数法。把小数变字符串 123 变成"123"
	// -1表示使用浮点数的默认精度进行格式化，不对小数部分进行特别的截断或四舍五入。意思就是保留小数点后几位数2 就是两位数
	// 64表示将value按照float64的精度来处理。
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// CreateOrderLimit 方法用于在币安期货交易所创建限价订单。限价订单 投资者设置一个合理的价格买卖 如btc60000 觉得太高了 设置4000再买
// 它接收订单的方向（买入或卖出）、交易对名称、交易数量和限价价格作为输入参数，并返回创建的订单信息或者错误。
func (b *BinanceFuture) CreateOrderLimit(side model.SideType, pair string,
	quantity float64, limit float64) (model.Order, error) {
	// 首先，使用 validate 方法校验提供的交易对和数量是否有效。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果校验失败，则返回一个空的 Order 对象和错误信息。
		return model.Order{}, err
	}

	// 使用客户端的 NewCreateOrderService 方法准备创建新的订单。
	order, err := b.client.NewCreateOrderService().
		Symbol(pair).                               // 设置订单的交易对。
		Type(futures.OrderTypeLimit).               // 设置订单类型为限价单。
		TimeInForce(futures.TimeInForceTypeGTC).    // 设置订单有效期为直到取消（Good Till Cancel）。
		Side(futures.SideType(side)).               // 设置订单方向为买入或卖出，根据参数 side 确定。
		Quantity(b.formatQuantity(pair, quantity)). // 设置订单数量，使用 formatQuantity 方法格式化。
		Price(b.formatPrice(pair, limit)).          // 设置订单价格，使用 formatPrice 方法格式化。
		Do(b.ctx)                                   // 调用 Do 方法并传入上下文，触发客户端向交易所发送订单请求。
	if err != nil {
		// 如果创建订单请求失败，则返回空的 Order 对象和错误信息。
		return model.Order{}, err
	}

	// 解析订单价格和原始数量的值。
	price, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		return model.Order{}, err
	}

	quantity, err = strconv.ParseFloat(order.OrigQuantity, 64)
	if err != nil {
		return model.Order{}, err
	}

	// 构造并返回一个 Order 对象，包含订单的详细信息。
	return model.Order{
		ExchangeID: order.OrderID,                                          // 订单在交易所的 ID。
		CreatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单创建时间，转换为毫秒级别，符合国际标准的时间格式。
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单最后更新时间。
		Pair:       pair,                                                   // 交易对。
		Side:       model.SideType(order.Side),                             // 订单方向（买/卖）。
		Type:       model.OrderType(order.Type),                            // 订单类型。
		Status:     model.OrderStatusType(order.Status),                    // 订单状态。
		Price:      price,                                                  // 订单价格。
		Quantity:   quantity,                                               // 订单数量。
	}, nil
}

// CreateOrderMarket 方法用于在币安期货交易所创建市价订单。市价订单是指以当前市场价格立即执行的订单，不限制价格。
// 这种类型的订单允许交易者立即买入或卖出资产，而无需等待价格达到特定水平。
func (b *BinanceFuture) CreateOrderMarket(side model.SideType, pair string, quantity float64) (model.Order, error) {
	err := b.validate(pair, quantity)
	if err != nil {
		return model.Order{}, err
	}

	order, err := b.client.NewCreateOrderService().
		Symbol(pair).
		Type(futures.OrderTypeMarket).
		Side(futures.SideType(side)).
		Quantity(b.formatQuantity(pair, quantity)).
		NewOrderResponseType(futures.NewOrderRespTypeRESULT). // 置订单响应类型为 RESULT 表示订单执行后立即返回执行结果，而不会等待其他条件的满足，如止损、止盈或限制条件
		Do(b.ctx)
	if err != nil {
		return model.Order{}, err
	}
	// 解析订单成交金额和已成交数量的值。返回这些数据可以让投资者了解订单的执行情况，包括已成交的数量和成交的总金额。这些信息对于投资者来说非常重要
	cost, err := strconv.ParseFloat(order.CumQuote, 64)
	if err != nil {
		return model.Order{}, err
	}

	quantity, err = strconv.ParseFloat(order.ExecutedQuantity, 64)
	if err != nil {
		return model.Order{}, err
	}

	// 返回一个包含订单详细信息的model.Order结构体。
	return model.Order{
		ExchangeID: order.OrderID,                                          // 订单在交易所的唯一标识符。
		CreatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单的创建时间，转换为毫秒级别的UNIX时间戳。
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单的最后更新时间，转换为毫秒级别的UNIX时间戳。
		Pair:       order.Symbol,                                           // order.Symbol 是订单的交易对标识符，表示交易对的基础货币和报价货币的组合
		Side:       model.SideType(order.Side),                             // 订单的交易方向（买入/卖出）。
		Type:       model.OrderType(order.Type),                            // 订单类型（市价/限价/止损/止盈等）。
		Status:     model.OrderStatusType(order.Status),                    // 订单的状态（已成交/未成交/部分成交等）。
		Price:      cost / quantity,                                        // 订单的成交均价，即成交总金额除以成交数量。
		Quantity:   quantity,                                               // 订单的数量。
	}, nil

}

// 假设投资者想要以市场最优价格购买比特币，但不愿意设置一个具体的购买价格。他可以使用市价报价订单来执行这个交易。投资者只需指定购买的数量，交易所将会以当前市场上最佳的价格立即执行这个订单。
func (b *BinanceFuture) CreateOrderMarketQuote(_ model.SideType, _ string, _ float64) (model.Order, error) {
	// 该功能没有实现
	panic("not implemented")
}

// 用于通过Binance Futures的API客户端向交易所发送取消订单的请求
func (b *BinanceFuture) Cancel(order model.Order) error {
	_, err := b.client.NewCancelOrderService().
		Symbol(order.Pair).        // 设置订单所属的交易对。
		OrderID(order.ExchangeID). // 设置要取消的订单在交易所的唯一标识符。
		Do(b.ctx)                  // 调用 Do 方法执行取消订单操作，并传入上下文对象 b.ctx。
	return err
}

// Orders 方法用于获取指定交易对的最新订单列表。
// 。函数 Orders 接收一个限制数量作为参数，并返回不超过这个数量的订单列表。如果存在超过限制数量的订单，它会返回前面的订单，否则返回所有订单。
func (b *BinanceFuture) Orders(pair string, limit int) ([]model.Order, error) {
	// 使用客户端的NewListOrdersService方法准备获取订单列表的请求。
	result, err := b.client.NewListOrdersService().
		Symbol(pair). // 设置订单所属的交易对。
		Limit(limit). // 设置要返回的订单数量的限制。
		Do(b.ctx)     // 执行获取订单列表的请求，并传入上下文对象 b.ctx。
	if err != nil {
		// 如果获取订单列表的请求失败，则直接返回错误。
		return nil, err
	}

	// 初始化一个空的订单列表。
	orders := make([]model.Order, 0)
	// 遍历获取到的订单结果列表，转换为自定义的订单结构并添加到订单列表中。
	for _, order := range result {
		orders = append(orders, newFutureOrder(order))
	}
	// 返回订单列表和 nil 错误，表示操作成功。
	return orders, nil
}

func (b *BinanceFuture) Order(pair string, id int64) (model.Order, error) {
	order, err := b.client.NewGetOrderService().
		Symbol(pair).
		OrderID(id).
		Do(b.ctx)

	if err != nil {
		return model.Order{}, err
	}

	return newFutureOrder(order), nil
}

// newFutureOrder 函数将来自 Binance 期货订单结构的数据转换为通用的订单模型。
// 使得该订单模型可以在任何系统中使用，而不仅仅局限于某个特定的交易所。通用订单模型可以提供一个统一的接口，使得系统可以处理来自不同交易所的订单数据，并进行统一的操作和管理。
// 如果订单的成交金额和已成交数量都大于零，则计算订单价格。
// 否则，使用订单的价格和原始数量计算价格。
func newFutureOrder(order *futures.Order) model.Order {
	var (
		price float64 // 订单价格
		err   error   // 错误
	)

	// 解析订单的累计成交金额和已成交数量。
	cost, _ := strconv.ParseFloat(order.CumQuote, 64)
	quantity, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)

	// 如果累计成交金额和已成交数量均大于零，则计算订单价格。
	if cost > 0 && quantity > 0 {
		price = cost / quantity
	} else {
		// 否则，使用订单的价格和原始数量计算价格。
		price, err = strconv.ParseFloat(order.Price, 64)
		log.CheckErr(log.WarnLevel, err) // 检查错误并记录警告级别日志。
		quantity, err = strconv.ParseFloat(order.OrigQuantity, 64)
		log.CheckErr(log.WarnLevel, err) // 检查错误并记录警告级别日志。
	}

	// 构造并返回一个通用订单模型。
	return model.Order{
		ExchangeID: order.OrderID,                                          // 订单在交易所的唯一标识符。
		Pair:       order.Symbol,                                           // 订单所属的交易对。
		CreatedAt:  time.Unix(0, order.Time*int64(time.Millisecond)),       // 订单创建时间。
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单最后更新时间。
		Side:       model.SideType(order.Side),                             // 订单方向（买/卖）。
		Type:       model.OrderType(order.Type),                            // 订单类型。
		Status:     model.OrderStatusType(order.Status),                    // 订单状态。
		Price:      price,                                                  // 订单价格。
		Quantity:   quantity,                                               // 订单数量。
	}
}

// BinanceFuture的Account方法用于获取账户信息，返回账户模型和错误信息。
func (b *BinanceFuture) Account() (model.Account, error) {
	// 使用Binance客户端的NewGetAccountService方法获取账户信息。
	acc, err := b.client.NewGetAccountService().Do(b.ctx)
	if err != nil {
		// 如果获取账户信息出错，则返回空的账户模型和错误信息。
		return model.Account{}, err
	}

	// 初始化一个空的Balance切片，用于存储账户的余额信息。
	balances := make([]model.Balance, 0)
	// 遍历账户的持仓信息。持仓信息关注于你对某个资产（比如BTC）的交易方向（做多或做空）以及相关的细节，比如持仓量和杠杆比率。
	for _, position := range acc.Positions {
		// 将持仓数量从字符串转换为float64类型。在期货交易中，当我们说到持仓数量，我们是指在特定合约或资产上的持有量，以及这些持有量是多头（看涨）还是空头（看跌）。
		free, err := strconv.ParseFloat(position.PositionAmt, 64)
		if err != nil {
			// 如果转换出错，则返回空的账户模型和错误信息。
			return model.Account{}, err
		}

		// 如果持仓数量为0，则跳过当前持仓。
		if free == 0 {
			continue
		}

		// 将杠杆比率从字符串转换为float64类型。如果你想进行价值1000美元的比特币交易，使用100倍的杠杆，你实际上只需要出资10美元。
		leverage, err := strconv.ParseFloat(position.Leverage, 64)
		if err != nil {
			// 如果转换出错，则返回空的账户模型和错误信息。
			return model.Account{}, err
		}

		// 如果持仓方向为做空，则将持仓数量设置为负值。负数是用来表示持有的是一个空头持仓（做空）
		if position.PositionSide == futures.PositionSideTypeShort {
			free = -free
		}

		// 分离标的资产和计价资产。例如交易对：ETHUSD 标的资产：ETH（以太坊）,计价资产：USD（美元）
		asset, _ := SplitAssetQuote(position.Symbol)

		// 将持仓信息添加到balances切片中。
		balances = append(balances, model.Balance{
			Asset:    asset,
			Free:     free,
			Leverage: leverage,
		})
	}

	// 遍历账户的资产信息。账户资产信息则提供了你账户中每种资产（不仅限于交易的标的资产，也包括计价资产和其他资产）的当前余额。
	for _, asset := range acc.Assets {
		// 将WalletBalance钱包余额从字符串转换为float64类型。
		free, err := strconv.ParseFloat(asset.WalletBalance, 64)
		if err != nil {
			// 如果转换出错，则返回空的账户模型和错误信息。
			return model.Account{}, err
		}

		// 如果钱包余额为0，则跳过当前资产。
		if free == 0 {
			continue
		}

		// 将资产信息添加到balances切片中。
		balances = append(balances, model.Balance{
			Asset: asset.Asset,
			Free:  free,
		})
	}

	// 构建并返回账户模型，这个字段是一个切片，包含所有余额信息。这个余额信息既包括了持仓信息（包括做多或做空的持仓量和相应的杠杆比率），也包括了账户资产信息（比如不同资产的总余额）。
	return model.Account{
		Balances: balances,
	}, nil
}

// Position 方法用于查询Binance Futures账户中特定交易对的基础资产和报价资产的总余额信息。
// 参数 pair 表示交易对的标识，如 "BTCUSDT"。
func (b *BinanceFuture) Position(pair string) (asset, quote float64, err error) {
	// 使用SplitAssetQuote函数分离交易对标识符，得到基础资产和报价资产的标识符。
	assetTick, quoteTick := SplitAssetQuote(pair)

	// 调用Account方法获取账户的全部余额信息。
	acc, err := b.Account()
	if err != nil {
		// 如果在获取账户信息时发生错误，返回错误。
		return 0, 0, err
	}

	// 使用Balance方法你可以查询账户里有多少个基础资产（如BTC）和报价资产（如USDT）
	assetBalance, quoteBalance := acc.Balance(assetTick, quoteTick)

	// 返回基础资产和报价资产的总余额（Free + Lock）。
	// Free代表可用余额，Lock代表当前被锁定（比如在未结算的订单中）的余额。
	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// CandlesSubscription 方法用于订阅特定交易对和周期的K线数据，并返回两个通道：一个用于接收K线数据，另一个用于接收错误信息。通过使用协程来处理futures.WsKlineServe函数的调用，你的程序可以在后台实时接收和处理K线数据，而不会干扰到主程序的其他操作。这种方式允许你的订阅实时更新，确保用户能够看到最新的数据。
func (b *BinanceFuture) CandlesSubscription(ctx context.Context, pair, period string) (chan model.Candle, chan error) {
	// 创建用于接收K线数据的通道 ccandle 和用于接收错误信息的通道 cerr。
	ccandle := make(chan model.Candle)
	cerr := make(chan error)

	// 创建 Heikin-Ashi 蜡烛图计算器。该计算器可用于将普通的K线数据转换为Heikin-Ashi平均蜡烛图数据。
	ha := model.NewHeikinAshi()

	go func() {
		// 创建指数退避器，当你在远程请求API获取数据时，如果发生错误，指数退避器会触发重试机制。但是要等到他设定的时间后你才能再次调用，以减轻对服务的负载压力
		ba := &backoff.Backoff{
			Min: 100 * time.Millisecond,
			Max: 1 * time.Second,
		}

		for {
			// futures.WsKlineServe函数通常是用来建立一个单向的数据流从Binance Futures交易所到机器人的WebSocket连接。，并在收到数据时将其发送到 ccandle 通道中。
			// 第三个参数是一个匿名函数，也称为回调函数。这个函数在每次有新的K线数据到来时被调用。函数的参数event是一个*futures.WsKlineEvent类型的指针，包含了最新的K线数据。
			done, _, err := futures.WsKlineServe(pair, period, func(event *futures.WsKlineEvent) {
				//这行代码调用了之前定义的指数退避器对象ba的Reset方法，用于重置退避计时。 每次成功接收数据后重置，以便下次重试时从最小等待时间开始。
				ba.Reset()
				// 将 WebSocket 接收到的原始K线数据转换为期货蜡烛图模型。
				candle := FutureCandleFromWsKline(pair, event.Kline)

				// 如果期货蜡烛图已经完整，并且设置了 Heikin-Ashi 蜡烛图选项，则将其转换为 Heikin-Ashi 蜡烛图。
				if candle.Complete && b.HeikinAshi {
					candle = candle.ToHeikinAshi(ha)
				}

				// 如果期货蜡烛图已经完整，则尝试获取附加数据并添加到蜡烛图的元数据中。
				if candle.Complete {
					// 遍历元数据提取器列表，因为MetadataFetchers是一个切片类型所以里面有很多个MetadataFetchers，所以遍历他得到fetcher 就符合单个MetadataFetchers类型，返回的字符串通常用作元数据的键（key），浮点数用作与该键相关联的值（value）。
					for _, fetcher := range b.MetadataFetchers {
						key, value := fetcher(pair, candle.Time)
						// 使用这对键值对更新蜡烛图对象中专门用于存储元数据的Metadata字段。
						candle.Metadata[key] = value
					}
				}

				// 将处理后的蜡烛图数据发送到 ccandle 通道中。
				ccandle <- candle
				// 第二个匿名函数如果在WebSocket订阅的过程中遇到错误，这个函数将错误发送到cerr通道中。
			}, func(err error) {
				cerr <- err
			})

			// futures.WsKlineServe函数尝试建立WebSocket连接并开始订阅时可能立即发生的错误。
			// 如果有错误 就把错误传入通道 然后关闭错误通道  同时关闭接收蜡烛图通道并结束
			if err != nil {
				cerr <- err
				close(cerr)
				close(ccandle)
				return
			}

			select {
			// 监听上下文的取消信号，这行代码监听ctx.Done()返回的通道。如果这个通道关闭了（意味着上下文被取消了）关闭cerr和ccandle两个通道，分别用于传递错误信息和蜡烛图数据。
			// ctx.Done()用于监听上下文（context）的取消信号。当你创建一个上下文对象时，你可以控制它，比如设置一个超时或手动取消。一旦上下文被取消（无论是因为超时、手动取消，还是其他原因），ctx.Done()返回的通道就会被关闭。
			case <-ctx.Done():
				close(cerr)
				close(ccandle)
				return
			// 监听 WebSocket 订阅是否已完成，：当done通道接收到信号时，执行case语句块内的代码time.Sleep(ba.Duration())：调用time.Sleep使当前goroutine暂停执行一段时间，这段时间由指数退避器ba的Duration方法确定。
			// done通道：done通道是在你的程序中自定义的，用于特定的同步信号，比如表示某个操作完成。在WebSocket订阅的例子中，它可能表示WebSocket连接已经关闭，或订阅者应该停止监听更多的数据。
			case <-done:
				time.Sleep(ba.Duration())
			}
		}
	}()

	// 返回用于接收K线数据的通道 ccandle 和用于接收错误信息的通道 cerr。
	return ccandle, cerr
}

// CandlesByLimit 获取指定交易对的最近一段时间内的K线数据，并限制返回的K线数量。
// 它接受一个上下文、交易对、时间周期和限制数量作为参数，返回K线数据切片和可能的错误。
func (b *BinanceFuture) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]model.Candle, error) {
	// 初始化一个空的K线数据切片。
	candles := make([]model.Candle, 0)
	// 创建一个用于获取K线数据的服务实例，该服务可以用于向币安期货交易所发送请求以获取指定交易对的K线数据。
	klineService := b.client.NewKlinesService()
	// 创通过调用 model.NewHeikinAshi() 来创建了一个 HeikinAshi 实例，这个实例可以用来处理和转换K线数据，以生成相应的 HeikinAshi 蜡烛图。，HeikinAshi 蜡烛图的计算方法包括利用前一个蜡烛图的平均值来计算当前蜡烛图的开盘价和收盘价。这种方法使得蜡烛图的价格走势更加平滑，有助于过滤噪音并更好地显示趋势方向
	ha := model.NewHeikinAshi()

	// 调用交易所的K线服务获取指定交易对的K线数据。Interval表示间隔或周期、Limit指的是限制k线的数量
	// Do(ctx) 可以比喻为一个遥控器，因为它是用来执行实际的请求的。通过调用 Do(ctx) 方法，实际的请求被发送到交易所，并等待获取相应的数据。
	data, err := klineService.Symbol(pair).
		Interval(period).
		Limit(limit + 1).
		Do(ctx)

	// 检查是否发生错误。
	if err != nil {
		return nil, err
	}

	// 遍历获取到的K线数据。
	for _, d := range data {
		// 将K线数据转换为 FutureCandle，并添加到K线数据切片中。拿到特定的交易对的k 线数据*d 传递一个指向 K 线数据的指针，可以在函数内部修改该 K 线数据的值。
		candle := FutureCandleFromKline(pair, *d)

		// 如果启用了 HeikinAshi，则将K线数据转换为 HeikinAshi 格式。
		if b.HeikinAshi {
			candle = candle.ToHeikinAshi(ha)
		}
		// 这行代码的作用是将转换后的蜡烛图数据(就是拿到特定交易对的蜡烛图数据) candle 添加到蜡烛图切片 candles 的末尾。
		candles = append(candles, candle)
	}

	// 删除最后一个不完整的K线数据，因为它可能是当前正在形成的K线。因为最后一根k线图有可能不完整。
	return candles[:len(candles)-1], nil
}

// CandlesByPeriod 方法根据指定的时间段获取K线数据。
func (b *BinanceFuture) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]model.Candle, error) {

	// 初始化一个空的蜡烛图数据切片。
	candles := make([]model.Candle, 0)
	// 创建K线服务的实例。
	klineService := b.client.NewKlinesService()
	// 创建一个Heikin Ashi蜡烛图的实例，用于后续可能的数据转换。
	ha := model.NewHeikinAshi()

	// 调用K线服务获取数据。你将从 Binance Futures API 获取到指定交易对（pair），在指定时间周期（period）内，从start时间到end时间范围内的K线数据
	// 如果时间间隔设置为"1h"（每小时），而请求的时间范围是24小时，那么理论上你将获取到24条K线数据
	data, err := klineService.Symbol(pair).
		Interval(period).
		StartTime(start.UnixNano() / int64(time.Millisecond)). //这段代码的作用确实是将开始时间的纳秒级别的时间戳转换成毫秒级别的时间戳。这个转换是必要的，因为 Binance API 要求时间参数以毫秒为单位。
		EndTime(end.UnixNano() / int64(time.Millisecond)).     //结束时间戳纳秒变成毫秒
		Do(ctx)

	// 如果请求出错，则返回错误。
	if err != nil {
		return nil, err
	}

	// 这一行遍历了从Binance Futures API获取到的K线数据。每一次迭代中，d代表了单条K线的数据，
	for _, d := range data {
		// 将原始K线数据转换为蜡烛图数据。
		candle := FutureCandleFromKline(pair, *d)

		// 如果设置了使用Heikin Ashi蜡烛图，则将普通蜡烛图数据转换为Heikin Ashi数据。
		if b.HeikinAshi {
			candle = candle.ToHeikinAshi(ha)
		}

		// 将转换后的蜡烛图数据添加到切片中。
		candles = append(candles, candle)
	}

	// 返回蜡烛图数据切片和nil表示没有错误。
	return candles, nil
}

// FutureCandleFromKline 从交易所K线数据创建一个 FutureCandle 结构。FutureCandle 结构表示了期货交易中的蜡烛图数据，它包含了一段时间内的开盘价、收盘价、最高价、最低价和交易量等信息。是将期货k线 变成蜡烛图更好的分析
func FutureCandleFromKline(pair string, k futures.Kline) model.Candle {
	// 参数 0 表示时间的起始点，开盘时间从毫秒级别转化为纳秒级别然后通过函数time.Unix 秒数和纳秒数转换为 time.Time 类型的时间对象如 2022-03-27 00:00:00 +0000 UTC 类型的时间对象默认以UTC（协调世界时）
	t := time.Unix(0, k.OpenTime*int64(time.Millisecond))

	// 创建一个 FutureCandle 结构，并初始化一些基本字段。
	candle := model.Candle{Pair: pair, Time: t, UpdatedAt: t}

	// 将K线数据中的开盘价、收盘价、最高价、最低价和交易量转换为浮点数，并赋值给相应的字段。
	var err error
	candle.Open, err = strconv.ParseFloat(k.Open, 64)
	log.CheckErr(log.WarnLevel, err)
	candle.Close, err = strconv.ParseFloat(k.Close, 64)
	log.CheckErr(log.WarnLevel, err)
	candle.High, err = strconv.ParseFloat(k.High, 64)
	log.CheckErr(log.WarnLevel, err)
	candle.Low, err = strconv.ParseFloat(k.Low, 64)
	log.CheckErr(log.WarnLevel, err)
	candle.Volume, err = strconv.ParseFloat(k.Volume, 64)
	log.CheckErr(log.WarnLevel, err)

	// 标记该蜡烛图为完整的。
	candle.Complete = true

	// 初始化一个空的元数据映射。
	candle.Metadata = make(map[string]float64)

	// 返回创建的 FutureCandle 结构。
	return candle
}

// 可以说这个方法的作用是把实时的期货K线数据转化成蜡烛图（K线图）的形式，以便于进行技术分析、策略回测或其他交易决策支持操作。
// 。这个函数FutureCandleFromWsKline是用于处理已经通过WebSocket连接接收到的K线数据。
// WebSocket 连接是一个双向连接 机器人与交易所之间的连接
func FutureCandleFromWsKline(pair string, k futures.WsKline) model.Candle {
	var err error // 用于捕获转换过程中可能发生的错误。

	// 将K线的开始时间（毫秒级时间戳）转换成time.Time类型，方便后续处理。
	t := time.Unix(0, k.StartTime*int64(time.Millisecond))

	// 初始化一个model.Candle结构体，设置交易对名称和时间。
	candle := model.Candle{Pair: pair, Time: t, UpdatedAt: t}

	// 将WebSocket K线数据中的开盘价、收盘价、最高价、最低价和成交量从字符串转换为浮点数。
	candle.Open, err = strconv.ParseFloat(k.Open, 64)     // 开盘价
	log.CheckErr(log.WarnLevel, err)                      // 检查并记录错误
	candle.Close, err = strconv.ParseFloat(k.Close, 64)   // 收盘价
	log.CheckErr(log.WarnLevel, err)                      // 检查并记录错误
	candle.High, err = strconv.ParseFloat(k.High, 64)     // 最高价
	log.CheckErr(log.WarnLevel, err)                      // 检查并记录错误
	candle.Low, err = strconv.ParseFloat(k.Low, 64)       // 最低价
	log.CheckErr(log.WarnLevel, err)                      // 检查并记录错误
	candle.Volume, err = strconv.ParseFloat(k.Volume, 64) // 成交量
	log.CheckErr(log.WarnLevel, err)                      // 检查并记录错误

	// 标记这个K线数据是否是该时间周期的最后一个数据点。
	candle.Complete = k.IsFinal

	// 初始化Metadata映射，用于存储可能的额外信息。
	candle.Metadata = make(map[string]float64)

	// 返回转换后的蜡烛图数据结构。
	return candle
}
