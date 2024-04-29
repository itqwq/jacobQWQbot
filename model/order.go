package model

import (
	"fmt"
	"time"
)

// SideType 定义订单的买卖方向，如买入或卖出。
type SideType string

// OrderType 定义订单的类型，如市价单或限价单等。
type OrderType string

// OrderStatusType 定义订单的状态，如新建、部分成交、完全成交等。
type OrderStatusType string

// 以下是 SideType 的可能值，表示买入或卖出。
var (
	SideTypeBuy  SideType = "BUY"
	SideTypeSell SideType = "SELL"
)

// 以下是 OrderType 的可能值，包括各种订单类型，如限价单、市价单等。
var (
	OrderTypeLimit           OrderType = "LIMIT"             // 限价单：买入限价单：如果一个投资者想要以不超过$39,000的价格购买比特币，他们可以下达一个以$39,000为限价的买入订单。这意味着，只有当比特币的价格降到$39,000或更低时，该买入订单才会被执行。买入也一样。
	OrderTypeMarket          OrderType = "MARKET"            // 市价单：订单将立即按照市场当前最优价格成交
	OrderTypeLimitMaker      OrderType = "LIMIT_MAKER"       // 限价挂单：确如果投资者下了一个以$35,000为价格的限价买入挂单，只有当市场价格下降到$35,000或更低时，这个订单才会被执行，市场价格一直高于$35,000，则该订单将保持未成交状态。
	OrderTypeStopLoss        OrderType = "STOP_LOSS"         // 止损单：当市场价格达到某个指定水平时自动执行，用于减少损失，止损市价单在触发止损价格（如$95）后会立即以市场上可用的最佳价格执行，不管这个价格是高于、等于还是低于止损价格。
	OrderTypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"   // 止损限价单：当市场价格达到触发价时，以限定的价格发送订单，如控制成交价格设置的限价是$94，那么即使止损条件（比如价格下跌到$95）被触发，订单也只会以$94或更高的价格成交，  但是如果突然跌倒93他就来不及成交了
	OrderTypeTakeProfit      OrderType = "TAKE_PROFIT"       // 获利单：当市场价格达到某个指定水平时自动执行，用于锁定盈利
	OrderTypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT" // 获利限价单：当前价格为$40,000 设置止盈触发价 45000 ，限价44500 ，当市场价格达到或超过止盈触发价$45,000时,订单才会被触发,找时机成交,如果来不及成交，最低可接收44500价格，如果还来不及接收，那么订单没有交易成功
)

// 以下是 OrderStatusType 的可能值，表示订单的各种状态。
var (
	OrderStatusTypeNew             OrderStatusType = "NEW"              // 新建订单：刚创建，尚未成交
	OrderStatusTypePartiallyFilled OrderStatusType = "PARTIALLY_FILLED" // 部分成交：订单只有部分数量成交
	OrderStatusTypeFilled          OrderStatusType = "FILLED"           // 完全成交：订单所有数量已成交
	OrderStatusTypeCanceled        OrderStatusType = "CANCELED"         // 已取消：订单已被用户或交易所取消
	OrderStatusTypePendingCancel   OrderStatusType = "PENDING_CANCEL"   // 取消中：取消订单的请求已提交，等待处理
	OrderStatusTypeRejected        OrderStatusType = "REJECTED"         // 已拒绝：订单因某些原因被交易所拒绝
	OrderStatusTypeExpired         OrderStatusType = "EXPIRED"          // 已过期：订单在成交前已过期
)

// Order 定义了订单的结构，包括订单的基本信息和状态。
type Order struct {
	ID         int64           `db:"id" json:"id" gorm:"primaryKey,autoIncrement"` // 订单ID，主键，自增
	ExchangeID int64           `db:"exchange_id" json:"exchange_id"`               // 交易所ID，标识订单所在的交易所
	Pair       string          `db:"pair" json:"pair"`                             // 交易对，如BTC/USD
	Side       SideType        `db:"side" json:"side"`                             // 订单方向，BUY 或 SELL
	Type       OrderType       `db:"type" json:"type"`                             // 订单类型，如LIMIT 或 MARKET
	Status     OrderStatusType `db:"status" json:"status"`                         // 订单状态，如NEW 或 FILLED
	Price      float64         `db:"price" json:"price"`                           // 订单价格
	Quantity   float64         `db:"quantity" json:"quantity"`                     // 订单数量

	CreatedAt time.Time `db:"created_at" json:"created_at"` // 订单创建时间
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"` // 订单最后更新时间

	// OCO (One Cancels the Other) 订单特有字段
	Stop    *float64 `db:"stop" json:"stop"`         // 止损价格
	GroupID *int64   `db:"group_id" json:"group_id"` // 订单组ID，用于将多个订单关联在一起

	// 以下字段仅用于内部使用，不持久化到数据库
	RefPrice    float64 `json:"ref_price" gorm:"-"`    // 参考价格，用于内部计算，列如在执行止损订单时，可能需要比较订单的止损价格与当前市场价格或参考价格来确定是否触发止损条件。
	Profit      float64 `json:"profit" gorm:"-"`       // 利润，用于内部计算
	ProfitValue float64 `json:"profit_value" gorm:"-"` // 利润价值，用于内部计算
	Candle      Candle  `json:"-" gorm:"-"`            // 关联的K线数据，用于内部分析，不持久化
}

// String 方法提供了订单信息的字符串表示，便于打印和记录。
func (o Order) String() string {
	return fmt.Sprintf("[%s] %s %s | ID: %d, Type: %s, %f x $%f (~$%.f)",
		o.Status, o.Side, o.Pair, o.ID, o.Type, o.Quantity, o.Price, o.Quantity*o.Price)
}
