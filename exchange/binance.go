package exchange

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
	"github.com/jpillora/backoff"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// MetadataFetchers 定义了一个函数类型，用于附加元数据到K线数据中。
// 它接受一个交易对和时间作为参数，并返回一个键（字符串）及其对应的值（浮点数）。
type MetadataFetchers func(pair string, t time.Time) (string, float64)

// Binance 结构体封装了与Binance交易所交互所需的配置和数据。
type Binance struct {
	ctx        context.Context            // ctx 提供了一个框架，用于发送取消信号、超时等控制指令给请求。
	client     *binance.Client            // client 是用于执行对Binance API请求的客户端实例。
	assetsInfo map[string]model.AssetInfo // assetsInfo 存储每个交易对的相关信息，如最小交易量和价格精度。
	HeikinAshi bool                       // HeikinAshi 标志用于指示是否将普通K线转换为Heikin Ashi K线。
	Testnet    bool                       // Testnet 标志用于指示是否使用Binance的测试网络进行API调用。

	APIKey    string // APIKey 用户的Binance API密钥，用于访问Binance API。
	APISecret string // APISecret 用户的Binance API密钥，用于访问Binance API。

	MetadataFetchers []MetadataFetchers // MetadataFetchers 是一个函数列表，用于在接收新K线数据后添加额外的元数据。
}

// BinanceOption 定义了一个函数类型，用于通过不同的配置选项来定制化Binance实例。
type BinanceOption func(*Binance)

// WithBinanceCredentials 返回一个BinanceOption函数，该函数设置Binance实例的API密钥和密钥。
func WithBinanceCredentials(key, secret string) BinanceOption {
	return func(b *Binance) {
		b.APIKey = key
		b.APISecret = secret
	}
}

// WithBinanceHeikinAshiCandle 返回一个BinanceOption函数，该函数启用Heikin Ashi蜡烛图转换。
func WithBinanceHeikinAshiCandle() BinanceOption {
	return func(b *Binance) {
		b.HeikinAshi = true
	}
}

// WithMetadataFetcher 配置一个元数据提取器，该提取器在接收新K线数据后运行，为K线数据的元数据添加额外信息。
func WithMetadataFetcher(fetcher MetadataFetchers) BinanceOption {
	return func(b *Binance) {
		// 将提取器函数添加到Binance实例的MetadataFetchers切片中，以便后续使用。
		b.MetadataFetchers = append(b.MetadataFetchers, fetcher)
	}
}

// WithTestNet 启用Binance的测试网络。
func WithTestNet() BinanceOption {
	return func(b *Binance) {
		// 设置Binance客户端使用测试网络。
		binance.UseTestnet = true
	}
}

// NewBinance 创建一个新的Binance实例，options 相当于type BinanceOption func(*Binance) 可以灵活的改动Binance结构体里面的任何东西 然后传参给NewBinance
func NewBinance(ctx context.Context, options ...BinanceOption) (*Binance, error) {
	// 开启WebSocket保持连接，以维持长时间的WebSocket连接不被断开。
	binance.WebsocketKeepalive = true

	// 初始化Binance结构体实例，设置上下文。
	exchange := &Binance{ctx: ctx}

	// 应用所有传入的配置选项，例如设置API密钥和启用Heikin Ashi蜡烛图等。
	for _, option := range options {
		option(exchange)
	}

	// 使用API密钥和密钥创建Binance API客户端。
	exchange.client = binance.NewClient(exchange.APIKey, exchange.APISecret)

	// 发送Ping请求到Binance服务器，检查API连接是否正常。
	err := exchange.client.NewPingService().Do(ctx)
	if err != nil {
		// 如果Ping失败，返回错误。
		return nil, fmt.Errorf("binance ping fail: %w", err)
	}

	// 获取交易所的交易对信息和交易限制，如最小交易量和价格精度。
	results, err := exchange.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化用于存储每个交易对资产信息的映射。
	exchange.assetsInfo = make(map[string]model.AssetInfo)
	// 遍历了从Binance交易所API获取的所有交易对信息(results.Symbols)。
	for _, info := range results.Symbols {
		tradeLimits := model.AssetInfo{
			BaseAsset:          info.BaseAsset,
			QuoteAsset:         info.QuoteAsset,
			BaseAssetPrecision: info.BaseAssetPrecision,
			QuotePrecision:     info.QuotePrecision,
		}
		// 遍历并设置交易对的交易限制，如最小交易量和步长。
		for _, filter := range info.Filters {
			if typ, ok := filter["filterType"]; ok {
				if typ == string(binance.SymbolFilterTypeLotSize) {
					tradeLimits.MinQuantity, _ = strconv.ParseFloat(filter["minQty"].(string), 64)
					tradeLimits.MaxQuantity, _ = strconv.ParseFloat(filter["maxQty"].(string), 64)
					tradeLimits.StepSize, _ = strconv.ParseFloat(filter["stepSize"].(string), 64)
				}
				if typ == string(binance.SymbolFilterTypePriceFilter) {
					tradeLimits.MinPrice, _ = strconv.ParseFloat(filter["minPrice"].(string), 64)
					tradeLimits.MaxPrice, _ = strconv.ParseFloat(filter["maxPrice"].(string), 64)
					tradeLimits.TickSize, _ = strconv.ParseFloat(filter["tickSize"].(string), 64)
				}
			}
		}
		// 将设置好的交易对资产信息添加到映射中。
		exchange.assetsInfo[info.Symbol] = tradeLimits
	}

	// 日志记录，表示Binance实例已成功设置并准备使用。
	log.Info("[SETUP] Using Binance exchange")

	// 返回配置好的Binance实例。
	return exchange, nil
}

// // LastQuote 获取指定交易对的最新报价。
func (b *Binance) LastQuote(ctx context.Context, pair string) (float64, error) {
	// 使用CandlesByLimit方法获取指定交易对最近1分钟的K线数据，限制数量为1，
	// 即只获取最新的一条K线数据。
	candles, err := b.CandlesByLimit(ctx, pair, "1m", 1)

	// 如果发生错误或未查询到任何K线数据（candles数组长度小于1），则返回错误。
	if err != nil || len(candles) < 1 {
		return 0, err
	}

	// 返回查询到的最新K线数据的收盘价。保证不管 limit 设置为多少 我拿到k线关闭价格是最新
	return candles[0].Close, nil
}

// 根据交易对拿到资产信息
func (b *Binance) AssetsInfo(pair string) model.AssetInfo {
	return b.assetsInfo[pair]
}

// 根据交易对，数量进行校验
func (b *Binance) validate(pair string, quantity float64) error {
	// 从assetsInfo映射中查找指定交易对的资产信息。
	info, ok := b.assetsInfo[pair]
	// 如果交易对不存在，则返回ErrInvalidAsset错误。
	if !ok {
		return ErrInvalidAsset
	}

	// 检查数量是否在交易所对该交易对设定的最小和最大数量限制之内。
	if quantity > info.MaxQuantity || quantity < info.MinQuantity {
		// 如果不符合，构建并返回一个包含错误详情的OrderError错误。
		return &OrderError{
			Err:      fmt.Errorf("%w: min: %f max: %f", ErrInvalidQuantity, info.MinQuantity, info.MaxQuantity),
			Pair:     pair,
			Quantity: quantity,
		}
	}

	// 如果数量符合限制，返回nil表示校验通过。
	return nil
}

// CreateOrderOCO 创建一个OCO（One Cancels the Other，一单成交另一单取消）订单。参数 买卖方向、交易对、数量、价格、止损价、止损限价
func (b *Binance) CreateOrderOCO(side model.SideType, pair string,
	quantity, price, stop, stopLimit float64) ([]model.Order, error) {

	// 验证给定的数量是否满足交易所的最小和最大交易量限制。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果校验失败，返回错误。
		return nil, err
	}

	// 使用Binance API客户端创建OCO订单。
	ocoOrder, err := b.client.NewCreateOCOService().
		Side(binance.SideType(side)).                     // 设置订单方向（买/卖）。
		Quantity(b.formatQuantity(pair, quantity)).       // 格式化并设置订单数量。
		Price(b.formatPrice(pair, price)).                // 格式化并设置订单价格。
		StopPrice(b.formatPrice(pair, stop)).             // 格式化并设置止损价格。
		StopLimitPrice(b.formatPrice(pair, stopLimit)).   // 格式化并设置止损限价。其止损价格为95美元 跌倒95美元触发止损价，但是不会立即执行，限价90美元：这意味着即使市场价格快速下跌，穿过了95美元，只要价格没有低于90美元，你的资产就有可能被出售。
		StopLimitTimeInForce(binance.TimeInForceTypeGTC). // 设置订单有效期为“直到取消”。根据市场的订单类型（如限价订单、市价订单）直到被成交，或者被用户手动取消。
		Symbol(pair).                                     // 设置交易对。
		Do(b.ctx)                                         // 执行创建订单的操作。
	if err != nil {
		// 如果创建订单失败，返回错误。
		return nil, err
	}

	// 初始化用于存储转换后订单信息的切片。这行代码创建了一个空的orders切片，准备用来存储类型为model.Order的元素，ocoOrder.Orders（从Binance API返回的订单列表）的长度
	orders := make([]model.Order, 0, len(ocoOrder.Orders))
	// 遍历从Binance返回的订单报告。OCOOrderReport是指在执行一个OCO（One Cancels the Other，一单成交另一单取消）订单操作后返回的订单报告。
	for _, order := range ocoOrder.OrderReports {
		// 转换订单价格和数量的类型。
		price, _ := strconv.ParseFloat(order.Price, 64)
		quantity, _ := strconv.ParseFloat(order.OrigQuantity, 64)
		// 构建内部订单模型。
		item := model.Order{
			ExchangeID: order.OrderID,                                                  // 订单在交易所的ID。
			CreatedAt:  time.Unix(0, ocoOrder.TransactionTime*int64(time.Millisecond)), // 订单创建时间。
			UpdatedAt:  time.Unix(0, ocoOrder.TransactionTime*int64(time.Millisecond)), // 订单更新时间。
			Pair:       pair,                                                           // 交易对。
			Side:       model.SideType(order.Side),                                     // 订单方向。
			Type:       model.OrderType(order.Type),                                    // 订单类型。
			Status:     model.OrderStatusType(order.Status),                            // 订单状态。
			Price:      price,                                                          // 订单价格。
			Quantity:   quantity,                                                       // 订单数量。
			GroupID:    &order.OrderListID,                                             // 订单组ID，用于将OCO订单联系起来。
		}

		// 如果订单是止损类型，设置止损价格。把item.Stop设置为stop的地址，让我们在需要的时候可以方便地查看或修改止损价格，同时也允许我们在不需要止损价格时，明确表示出这一点。这种方式既节省了资源，又提高了程序处理的灵活性。想拿到值 *item.Stop
		if item.Type == model.OrderTypeStopLossLimit || item.Type == model.OrderTypeStopLoss {
			item.Stop = &stop
		}

		// 将订单添加到结果切片中。
		orders = append(orders, item)
	}

	// 返回创建的订单和nil（表示没有错误）。
	return orders, nil
}

// 定义CreateOrderStop方法，用于创建止损卖单。参数 交易对、 数量、止损价格
func (b *Binance) CreateOrderStop(pair string, quantity float64, limit float64) (model.Order, error) {
	// 验证交易对和数量是否符合交易所的要求。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果验证失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 使用Binance API客户端创建新的订单服务，配置订单参数。
	order, err := b.client.NewCreateOrderService().Symbol(pair).
		Type(binance.OrderTypeStopLoss).            // 设置订单类型为止损。
		TimeInForce(binance.TimeInForceTypeGTC).    // 设置订单有效期为直到取消（GTC）。
		Side(binance.SideTypeSell).                 // 设置订单为卖出。
		Quantity(b.formatQuantity(pair, quantity)). // 设置订单数量，经过格式化。
		Price(b.formatPrice(pair, limit)).          // 设置止损价格，经过格式化。
		Do(b.ctx)                                   // 执行订单创建操作。
	if err != nil {
		// 如果创建订单失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 解析返回的订单价格和原始数量。
	price, _ := strconv.ParseFloat(order.Price, 64)
	quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)

	// 构造并返回一个填充了订单信息的Order结构体。
	return model.Order{
		ExchangeID: order.OrderID,                                            // 交易所的订单ID。
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单创建时间，转换为Go的时间格式。
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单更新时间，同上。
		Pair:       pair,                                                     // 交易对。
		Side:       model.SideType(order.Side),                               // 订单方向（卖出）。
		Type:       model.OrderType(order.Type),                              // 订单类型（止损）。
		Status:     model.OrderStatusType(order.Status),                      // 订单状态。
		Price:      price,                                                    // 订单价格。
		Quantity:   quantity,                                                 // 订单数量。
	}, nil
}

// formatPrice用于格式化给定的价格值，确保它符合特定交易对的价格规则。
func (b *Binance) formatPrice(pair string, value float64) string {
	// 尝试从b.assetsInfo中获取指定交易对的资产信息。
	if info, ok := b.assetsInfo[pair]; ok {
		// 这里info.TickSize表示价格的最小变动单位，info.QuotePrecision表示报价的精度。
		value = common.AmountToLotSize(info.TickSize, info.QuotePrecision, value)
	}
	// 使用strconv.FormatFloat将调整精度后的value转换为字符串。
	// 'f'指定格式化的方式（不使用科学计数法），-1表示根据实际情况自动选择小数点后的位数，64表示处理为float64类型的值。
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func (b *Binance) formatQuantity(pair string, value float64) string {
	if info, ok := b.assetsInfo[pair]; ok {
		value = common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, value)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// CreateOrderLimit 创建一个限价订单。
func (b *Binance) CreateOrderLimit(side model.SideType, pair string,
	quantity float64, limit float64) (model.Order, error) {

	// 验证交易对和数量是否符合交易所的规则。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果验证失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 使用Binance API客户端配置并发送创建限价订单的请求。
	order, err := b.client.NewCreateOrderService().
		Symbol(pair).                               // 设置交易对。
		Type(binance.OrderTypeLimit).               // 设置订单类型为限价。
		TimeInForce(binance.TimeInForceTypeGTC).    // 设置订单有效期为“直到取消（GTC）”。
		Side(binance.SideType(side)).               // 设置订单买卖方向（买入/卖出）。
		Quantity(b.formatQuantity(pair, quantity)). // 设置订单数量，经过格式化以符合交易对要求。
		Price(b.formatPrice(pair, limit)).          // 设置订单价格，经过格式化以符合交易对要求。
		Do(b.ctx)                                   // 执行创建订单操作。
	if err != nil {
		// 如果创建订单失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 解析返回的订单价格，确保其格式正确。
	price, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		// 如果价格格式转换失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 解析返回的订单原始数量，确保其格式正确。
	quantity, err = strconv.ParseFloat(order.OrigQuantity, 64)
	if err != nil {
		// 如果数量格式转换失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 构造并返回一个填充了订单信息的Order结构体实例。
	return model.Order{
		ExchangeID: order.OrderID,                                            // 订单在交易所的唯一标识符。
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单创建时间。
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单更新时间。
		Pair:       pair,                                                     // 交易对。
		Side:       model.SideType(order.Side),                               // 订单买卖方向。
		Type:       model.OrderType(order.Type),                              // 订单类型（限价）。
		Status:     model.OrderStatusType(order.Status),                      // 订单状态。
		Price:      price,                                                    // 订单价格。
		Quantity:   quantity,                                                 // 订单数量。
	}, nil
}

// CreateOrderMarket 创建一个市价订单。CreateOrderMarket 方法直接指定了要购买或出售的货币(如btc 1个)数量，
func (b *Binance) CreateOrderMarket(side model.SideType, pair string, quantity float64) (model.Order, error) {
	// 首先，验证交易对和数量是否符合交易所的规则。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果验证失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 使用Binance API客户端配置并发送创建市价订单的请求。
	order, err := b.client.NewCreateOrderService().
		Symbol(pair).                                   // 设置交易对。
		Type(binance.OrderTypeMarket).                  // 设置订单类型为市价。
		Side(binance.SideType(side)).                   // 设置订单买卖方向（买入/卖出）。
		Quantity(b.formatQuantity(pair, quantity)).     // 设置订单数量，经过格式化以符合交易对要求。
		NewOrderRespType(binance.NewOrderRespTypeFULL). // 设置返回类型为完整响应。味着你希望在订单创建成功后，API返回包含完整订单信息的响应。这包括订单的所有细节，如成交量、成交价格、订单状态等。
		Do(b.ctx)                                       // 执行创建订单操作。
	if err != nil {
		// 如果创建订单失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 解析返回的累计报价数量（成交额）所有成交的累计金额。市价订单时，所有成交的累计金额，也就是交易的总成本或总收入
	cost, err := strconv.ParseFloat(order.CummulativeQuoteQuantity, 64)
	if err != nil {
		return model.Order{}, err
	}

	// 解析返回的实际执行数量，确保其格式正确。这表示市价订单在市场上实际成交的资产数量
	quantity, err = strconv.ParseFloat(order.ExecutedQuantity, 64)
	if err != nil {
		return model.Order{}, err
	}

	// 构造并返回一个填充了订单信息的Order结构体实例。
	return model.Order{
		ExchangeID: order.OrderID,                                            // 订单在交易所的唯一标识符。
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单创建时间。
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单更新时间。
		Pair:       order.Symbol,                                             // 交易对。
		Side:       model.SideType(order.Side),                               // 订单买卖方向。
		Type:       model.OrderType(order.Type),                              // 订单类型（市价）。
		Status:     model.OrderStatusType(order.Status),                      // 订单状态。
		Price:      cost / quantity,                                          // 计算平均成交价格。
		Quantity:   quantity,                                                 // 订单数量。
	}, nil
}

// CreateOrderMarketQuote 创建一个基于报价金额的市价订单。将交易对设置为BTC/USDT，然后指定报价货币数量为100 USDT，这样就会以市价条件购买BTC，获得相应的BTC数量。而 CreateOrderMarketQuote 方法则指定了用于交易的报价金额（通常是基准货币的数量）。
func (b *Binance) CreateOrderMarketQuote(side model.SideType, pair string, quantity float64) (model.Order, error) {
	// 验证交易对和报价数量是否符合交易所的规则。
	err := b.validate(pair, quantity)
	if err != nil {
		// 如果验证失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 使用Binance API客户端配置并发送创建市价订单的请求，这里的订单是基于报价金额而不是数量。
	order, err := b.client.NewCreateOrderService().
		Symbol(pair).                                    // 设置交易对。
		Type(binance.OrderTypeMarket).                   // 设置订单类型为市价。
		Side(binance.SideType(side)).                    // 设置订单买卖方向（买入/卖出）。
		QuoteOrderQty(b.formatQuantity(pair, quantity)). // 设置基于报价金额（USDT）的订单数量，经过格式化以符合交易对要求。
		NewOrderRespType(binance.NewOrderRespTypeFULL).  // 设置返回类型为完整响应。
		Do(b.ctx)                                        // 执行创建订单操作。
	if err != nil {
		// 如果创建订单失败，返回空的Order结构体和错误信息。
		return model.Order{}, err
	}

	// 解析返回的累计报价数量（成交额），确保其格式正确。
	cost, err := strconv.ParseFloat(order.CummulativeQuoteQuantity, 64)
	if err != nil {
		return model.Order{}, err
	}

	// 解析返回的实际执行数量，确保其格式正确。
	quantity, err = strconv.ParseFloat(order.ExecutedQuantity, 64)
	if err != nil {
		return model.Order{}, err
	}

	// 构造并返回一个填充了订单信息的Order结构体实例。
	return model.Order{
		ExchangeID: order.OrderID,                                            // 订单在交易所的唯一标识符。
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单创建时间。
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)), // 订单更新时间。
		Pair:       order.Symbol,                                             // 交易对。
		Side:       model.SideType(order.Side),                               // 订单买卖方向。
		Type:       model.OrderType(order.Type),                              // 订单类型（市价）。
		Status:     model.OrderStatusType(order.Status),                      // 订单状态。
		Price:      cost / quantity,                                          // 计算平均成交价格。
		Quantity:   quantity,                                                 // 实际执行数量。
	}, nil
}

func (b *Binance) Cancel(order model.Order) error {
	// 使用Binance API客户端配置并发送取消订单的请求，指定要取消的订单的交易对和订单ID。
	_, err := b.client.NewCancelOrderService().
		Symbol(order.Pair).        // 设置订单的交易对。
		OrderID(order.ExchangeID). // 设置要取消的订单的交易所ID。
		Do(b.ctx)                  // 执行取消订单操作。

	// 返回取消订单操作的结果（nil表示成功，否则表示失败）。
	return err
}

// Orders 方法用于获取指定交易对的订单列表。pair：要获取订单列表的交易对，limit：返回的订单数量限制。
func (b *Binance) Orders(pair string, limit int) ([]model.Order, error) {
	// 使用 Binance API 客户端配置并发送获取订单列表的请求，指定交易对和订单数量限制。
	result, err := b.client.NewListOrdersService().
		Symbol(pair). // 设置要获取订单的交易对。
		Limit(limit). // 设置返回的订单数量限制。
		Do(b.ctx)     // 执行获取订单列表的操作。
	if err != nil {
		// 如果获取订单列表失败，返回空的订单切片和错误信息。
		return nil, err
	}

	// 初始化一个空的订单切片，用于存储转换后的订单信息。
	orders := make([]model.Order, 0)
	// 遍历从 Binance API 返回的订单列表。
	for _, order := range result {
		// 将每个订单转换为内部模型并添加到订单切片中。
		orders = append(orders, newOrder(order))
	}
	// 返回获取到的订单列表和 nil（表示没有错误）。
	return orders, nil
}

// Order，用于从 Binance 交易所获取指定交易对和订单ID的订单信息。
func (b *Binance) Order(pair string, id int64) (model.Order, error) {
	// 使用 Binance 客户端的 NewGetOrderService 方法创建一个获取订单信息的服务。
	// 调用 Symbol 方法设置订单的交易对。
	// 调用 OrderID 方法设置订单的唯一标识符。
	// 调用 Do 方法发送请求并获取订单信息，该方法接受一个上下文参数 b.ctx。
	order, err := b.client.NewGetOrderService().
		Symbol(pair).
		OrderID(id).
		Do(b.ctx)

	// 如果在获取订单信息时发生了错误，直接返回空订单和错误信息。
	if err != nil {
		return model.Order{}, err
	}

	// 将获取到的订单信息转换为应用内部的订单模型，并返回给调用者。
	return newOrder(order), nil
}

// newOrder 函数用于将从 Binance API 返回的订单转换为内部订单模型。
// 它接收一个指向 binance.Order 类型的指针作为参数，并返回一个 model.Order 类型的实例。
func newOrder(order *binance.Order) model.Order {
	var price float64

	// 解析订单的累计报价数量（成交额）和实际执行数量，并计算平均成交价格。
	cost, _ := strconv.ParseFloat(order.CummulativeQuoteQuantity, 64)
	quantity, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)
	if cost > 0 && quantity > 0 {
		price = cost / quantity
	} else {
		// 如果累计成交额或实际成交数量不大于0，则解析订单价格和原始数量。
		price, _ = strconv.ParseFloat(order.Price, 64)
		quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)
	}

	// 构建并返回内部订单模型。
	return model.Order{
		ExchangeID: order.OrderID,                                          // 订单在交易所的唯一标识符。
		Pair:       order.Symbol,                                           // 交易对。
		CreatedAt:  time.Unix(0, order.Time*int64(time.Millisecond)),       // 订单创建时间。
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)), // 订单更新时间。
		Side:       model.SideType(order.Side),                             // 订单买卖方向。
		Type:       model.OrderType(order.Type),                            // 订单类型。
		Status:     model.OrderStatusType(order.Status),                    // 订单状态。
		Price:      price,                                                  // 订单价格。
		Quantity:   quantity,                                               // 订单数量。
	}
}

// Account 方法用于获取用户在交易所的账户信息。仅用于获取用户的现货账户信息，因此没有返回持仓信息
func (b *Binance) Account() (model.Account, error) {
	// 使用Binance API客户端获取账户信息。
	acc, err := b.client.NewGetAccountService().Do(b.ctx)
	if err != nil {
		// 如果获取账户信息失败，则返回空的Account结构体和错误信息。
		return model.Account{}, err
	}

	// 初始化一个用于存储用户资产余额的切片。
	balances := make([]model.Balance, 0)
	// 遍历从Binance返回的账户余额信息。
	for _, balance := range acc.Balances {
		// 解析账户中资产的可用余额和冻结余额。
		free, err := strconv.ParseFloat(balance.Free, 64)
		if err != nil {
			return model.Account{}, err
		}
		locked, err := strconv.ParseFloat(balance.Locked, 64)
		if err != nil {
			return model.Account{}, err
		}
		// 将解析后的资产余额信息添加到balances切片中。
		balances = append(balances, model.Balance{
			Asset: balance.Asset, // 资产名称。
			Free:  free,          // 可用余额。
			Lock:  locked,        // 冻结余额。
		})
	}

	// 构建并返回包含账户资产余额信息的Account结构体实例。
	return model.Account{
		Balances: balances, // 用户的资产余额信息。
	}, nil
}

// Position 方法确实用于查询指定交易对的基础资产和报价资产的总余额。
// 参数 pair 指定了要查询的交易对。
func (b *Binance) Position(pair string) (asset, quote float64, err error) {
	// 使用 SplitAssetQuote 函数将交易对拆分为资产货币和报价货币。
	assetTick, quoteTick := SplitAssetQuote(pair)
	// 调用 Account 方法获取用户在交易所的账户信息。
	acc, err := b.Account()
	if err != nil {
		return 0, 0, err
	}

	// 从账户信息中获取指定资产货币和报价货币的余额。
	assetBalance, quoteBalance := acc.Balance(assetTick, quoteTick)

	// 计算资产货币和报价货币的总余额（包括可用和冻结）。
	assetTotal := assetBalance.Free + assetBalance.Lock
	quoteTotal := quoteBalance.Free + quoteBalance.Lock

	return assetTotal, quoteTotal, nil
}

// CandlesSubscription 方法用于订阅指定交易对和周期的K线数据流，并返回两个通道，一个用于接收K线数据，另一个用于接收错误信息。
// ctx 是上下文对象，用于控制订阅的生命周期。
// pair 是交易对，表示要订阅的资产对，例如 "BTCUSDT"。
// period 是K线周期，例如 "1m" 表示1分钟周期，"1h" 表示1小时周期。
// 返回值 ccandle 是一个通道，用于接收 K 线数据。
// 返回值 cerr 是一个通道，用于接收订阅过程中的错误信息。
func (b *Binance) CandlesSubscription(ctx context.Context, pair, period string) (chan model.Candle, chan error) {
	// 创建用于传递 K 线数据的通道。
	ccandle := make(chan model.Candle)
	// 创建用于传递错误信息的通道。
	cerr := make(chan error)
	// 创建 Heikin Ashi 指标实例。
	ha := model.NewHeikinAshi()

	go func() {
		// 创建指数退避器，用于处理连接失败后的重试机制。
		ba := &backoff.Backoff{
			Min: 100 * time.Millisecond,
			Max: 1 * time.Second,
		}

		// 循环进行 K 线数据订阅，直到订阅被取消或发生错误。
		for {
			// 启动 K 线数据订阅，返回一个 done 通道用于信号订阅是否完成，以及一个错误通道用于传递连接错误。
			done, _, err := binance.WsKlineServe(pair, period, func(event *binance.WsKlineEvent) {
				// 重置指数退避器，以便在成功接收数据时重新计时。
				ba.Reset()
				// 将 WebSocket 事件转换为 Candle 结构。
				candle := CandleFromWsKline(pair, event.Kline)

				// 如果 K 线数据完整并且启用了 Heikin Ashi，则转换为 Heikin Ashi 格式。
				if candle.Complete && b.HeikinAshi {
					candle = candle.ToHeikinAshi(ha)
				}

				// 如果 K 线数据完整，则尝试获取额外的元数据。
				if candle.Complete {
					// 遍历元数据获取器列表，获取 K 线时间点的附加数据。
					for _, fetcher := range b.MetadataFetchers {
						key, value := fetcher(pair, candle.Time)
						candle.Metadata[key] = value
					}
				}

				// 将处理后的 K 线数据发送到通道中。
				ccandle <- candle
			}, func(err error) {
				// 在发生连接错误时，将错误信息发送到错误通道中。
				cerr <- err
			})
			// 如果发生连接错误，则将错误信息发送到错误通道中并关闭通道，然后退出循环。
			if err != nil {
				cerr <- err
				close(cerr)
				close(ccandle)
				return
			}

			// 等待 ctx 取消信号或订阅完成信号，如果收到取消信号，则关闭通道并退出循环。
			select {
			case <-ctx.Done():
				close(cerr)
				close(ccandle)
				return
			case <-done:
				// 如果收到订阅完成信号，则根据指数退避器的计时等待一段时间后重试订阅。
				time.Sleep(ba.Duration())
			}
		}
	}()

	// 返回 K 线数据通道和错误通道。
	return ccandle, cerr
}

// CandlesByLimit 获取指定交易对、时间周期和数量限制的K线数据。
func (b *Binance) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]model.Candle, error) {
	// 初始化用于存储K线数据的切片。
	candles := make([]model.Candle, 0)
	// 创建K线服务实例。
	klineService := b.client.NewKlinesService()
	// 初始化Heikin Ashi蜡烛图计算器，用于可能的数据转换。
	ha := model.NewHeikinAshi()

	// 从Binance API获取K线数据，请求的数量比实际需要的多1，
	// 因为最后一根K线可能是不完整的，需要丢弃。
	data, err := klineService.Symbol(pair).
		Interval(period).
		Limit(limit + 1). //加一是保证有完整的k线图
		Do(ctx)

	// 如果请求过程中发生错误，则返回错误。
	if err != nil {
		return nil, err
	}

	// 遍历获取到的K线数据，将每条数据转换为model.Candle格式。
	for _, d := range data {
		candle := CandleFromKline(pair, *d)

		// 如果启用了Heikin Ashi蜡烛图转换，则对数据进行转换。
		if b.HeikinAshi {
			candle = candle.ToHeikinAshi(ha)
		}

		// 将转换好的K线数据添加到结果切片中。
		candles = append(candles, candle)
	}

	// 丢弃最后一根可能不完整的K线，返回剩余的K线数据。
	return candles[:len(candles)-1], nil
}

// CandlesByPeriod 方法用于获取指定交易对、周期、时间范围内的 K 线数据。
func (b *Binance) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]model.Candle, error) {
	// 初始化用于存储 K 线数据的切片。
	candles := make([]model.Candle, 0)
	// 创建 K 线查询服务实例。
	klineService := b.client.NewKlinesService()
	// 创建 Heikin Ashi 指标实例。
	ha := model.NewHeikinAshi()

	// 发起 K 线查询请求，获取指定交易对、周期、时间范围内的原始 K 线数据。
	data, err := klineService.Symbol(pair).
		Interval(period).
		StartTime(start.UnixNano() / int64(time.Millisecond)).
		EndTime(end.UnixNano() / int64(time.Millisecond)).
		Do(ctx)

	// 如果查询过程中发生错误，则返回错误信息。
	if err != nil {
		return nil, err
	}

	// 遍历返回的原始 K 线数据，并将其转换为应用层的 Candle 结构。
	for _, d := range data {
		// 将原始 K 线数据转换为 Candle 结构。
		candle := CandleFromKline(pair, *d)

		// 如果启用了 Heikin Ashi，则将 K 线数据转换为 Heikin Ashi 格式。
		if b.HeikinAshi {
			candle = candle.ToHeikinAshi(ha)
		}

		// 将处理后的 Candle 结构添加到切片中。
		candles = append(candles, candle)
	}

	// 返回处理后的 K 线数据切片。
	return candles, nil
}

// 把拿到的k线数据存入结构体 1. 变成蜡烛图  2.更好的保存数据
// 这个函数通常用于处理历史K线数据的查询结果，例如从交易所的REST API获取历史K线数据。
// CandleFromKline 将binance.Kline类型的数据转换为model.Candle类型。
func CandleFromKline(pair string, k binance.Kline) model.Candle {
	// 将K线的开盘时间从毫秒转换为time.Time类型，用于K线的时间标记。
	// 第一个参数是秒数，由于k.OpenTime以毫秒为单位，所以这里将秒数设置为0
	// 第二个参数表示纳秒数，k.OpenTime是以一个毫秒为单位的时间戳，将k.OpenTime乘以time.Millisecond，将毫秒转化为纳秒，转化为int64类型
	//将以毫秒为单位的时间戳 k.OpenTime 转换为对应的 time.Time 时间表示，确实是通过将其转化为纳秒来实现的。
	t := time.Unix(0, k.OpenTime*int64(time.Millisecond))

	// 初始化candle结构，设置交易对、时间和更新时间。
	candle := model.Candle{Pair: pair, Time: t, UpdatedAt: t}

	// 转换字符串格式的开盘、收盘、最高、最低价格和成交量到float64类型。
	candle.Open, _ = strconv.ParseFloat(k.Open, 64)
	candle.Close, _ = strconv.ParseFloat(k.Close, 64)
	candle.High, _ = strconv.ParseFloat(k.High, 64)
	candle.Low, _ = strconv.ParseFloat(k.Low, 64)
	candle.Volume, _ = strconv.ParseFloat(k.Volume, 64)

	// 标记此K线数据为完整的。
	candle.Complete = true

	// 初始化candle的元数据映射。
	candle.Metadata = make(map[string]float64)

	// 返回转换后的candle结构。
	return candle
}

// CandleFromWsKline 函数用于从 WebSocket 接收的 K 线数据创建 Candle 结构。
// 这个函数通常用于处理实时更新的K线数据，例如通过WebSocket实时订阅的K线数据。
func CandleFromWsKline(pair string, k binance.WsKline) model.Candle {
	// 解析 K 线数据中的时间戳，并转换为 Go 时间对象。
	t := time.Unix(0, k.StartTime*int64(time.Millisecond))
	// 初始化 Candle 结构，设置交易对和时间。
	candle := model.Candle{Pair: pair, Time: t, UpdatedAt: t}
	// 解析 K 线数据中的开盘价、收盘价、最高价、最低价、交易量等信息，并填充到 Candle 结构中。
	candle.Open, _ = strconv.ParseFloat(k.Open, 64)
	candle.Close, _ = strconv.ParseFloat(k.Close, 64)
	candle.High, _ = strconv.ParseFloat(k.High, 64)
	candle.Low, _ = strconv.ParseFloat(k.Low, 64)
	candle.Volume, _ = strconv.ParseFloat(k.Volume, 64)
	candle.Complete = k.IsFinal                // 设置 Candle 的完整标志。
	candle.Metadata = make(map[string]float64) // 初始化元数据字段。
	// 返回填充了 K 线数据的 Candle 结构。
	return candle
}
