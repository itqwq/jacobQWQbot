// package service 定义了与金融市场交易所交互的接口和结构。
// 它包括了市场数据消费、交易操作和通知的抽象。
package service

import (
	"context" // 为管理进程和操作的生命周期提供原语。
	"time"    // 用于处理与时间相关的操作，如K线的开始和结束时间。

	"github.com/rodrigo-brito/ninjabot/model" // 导入model包，该包含有交易相关的数据结构。
)

// Exchange 接口结合了Broker和Feeder接口，
// 代表了一个与金融交易所交互的全功能接口。
type Exchange interface {
	Broker // 用于执行和管理交易的接口。
	Feeder // 用于接收市场数据的接口。
}

// Feeder 接口提供了获取市场数据的方法，如交易对信息、
// 实时报价、历史K线数据，以及订阅实时K线更新。
type Feeder interface {
	AssetsInfo(pair string) model.AssetInfo                      // 获取交易对的信息。
	LastQuote(ctx context.Context, pair string) (float64, error) // 获取某交易对的最后成交价格。
	// 按指定周期和时间范围检索K线数据。就是一段时间的k线图 传入上下文，交易对，时间周期period 比如"1h" 开始时间，结束时间， 返回一个蜡烛图的k线信息
	CandlesByPeriod(ctx context.Context, pair, period string, start, end time.Time) ([]model.Candle, error)
	CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]model.Candle, error)      // 按限制数量检索K线数据。limit 限制k线的数量 比如 limit=10  交易对btc  1h 然后就是最近十小时btc返回的数据
	CandlesSubscription(ctx context.Context, pair, timeframe string) (chan model.Candle, chan error) // 订阅实时K线更新。timeframe 指的是k线的价格动态常见的timeframe包括："1m"：每分钟，每根K线代表一分钟内的价格动态。	"5m"：每5分钟，每根K线代表五分钟内的价格动态

}

// Broker 接口提供了执行交易、管理订单等功能的方法。
type Broker interface {
	Account() (model.Account, error)                        // 获取账户信息。
	Position(pair string) (asset, quote float64, err error) // 获取某交易对的持仓信息。
	Order(pair string, id int64) (model.Order, error)       // 获取指定订单的详细信息。
	// 创建OCO（一单成交即取消另一单）订单。置两个不一样的价格一个是获利 一个是止损 只要触发哪一个 订单马上取消 size 数量 比如 btc 1个  price 获利价 stop 止损我们设置得价格, 到了stop 开始执行stopLimit 止损单执行价格
	CreateOrderOCO(side model.SideType, pair string, size, price, stop, stopLimit float64) ([]model.Order, error)
	CreateOrderLimit(side model.SideType, pair string, size float64, limit float64) (model.Order, error) // 创建限价订单。他们愿意交易（买入或卖出）的最优价格。当市场价格达到或更优于这个价格时，订单会被执行。。limit 1000比如 btc 1000 买入
	CreateOrderMarket(side model.SideType, pair string, size float64) (model.Order, error)               // 创建市价订单。市价订单是一种立即按当前市场价格执行的订单类型，与限价订单相反，它不允许交易者指定成交价格。这个方法的目的是让交易者能够快速进入或退出市场，通常用于需要立即成交的场景。
	CreateOrderMarketQuote(side model.SideType, pair string, quote float64) (model.Order, error)         // 以报价金额创建市价订单。比如我的账户有100000 我想出1000买btc 意思就是说这个方法可以以自己想买的金额买入
	CreateOrderStop(pair string, quantity float64, limit float64) (model.Order, error)                   // 创建止损订单。，旨在限制投资者的损失。当交易资产的价格达到或者超过某个指定的价格点（止损价）时，止损订单会被触发，自动以市价或限价卖出（或买入，如果是做空操作）该资产。
	Cancel(model.Order) error                                                                            // 取消订单。
}

// Notifier 接口定义了通知相关的方法。
type Notifier interface {
	Notify(string)             // 发送通知消息。可以用于广泛的通知需求，如交易确认、重要市场更新、系统消息等。
	OnOrder(order model.Order) // 当订单事件发生时的回调。当一个订单事件发生时（例如订单被创建、执行或取消），这个方法会被调用。
	OnError(err error)         // 当错误发生时的回调。，当交易失败、数据加载出错或系统内部发生异常时，通过这个方法可以通知用户或记录错误日志。
}

// Telegram 接口继承了Notifier接口，添加了启动方法。
type Telegram interface {
	Notifier // 继承Notifier接口。
	Start()  // 启动通知服务。Start()方法: 是Telegram接口特有的方法，用于启动通知服务。具体来说，这可能涉及到进行初始化操作、建立网络连接、准备发送消息的资源等，以便通知服务能够正常运行。
}
