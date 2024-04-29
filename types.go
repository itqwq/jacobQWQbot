package ninjabot

import (
	"github.com/rodrigo-brito/ninjabot/model"
)

/*
这段代码主要集合了与交易相关的核心变量和类型，使得在编写与交易相关的功能时，可以更方便地引用这些变量和类型。这包括订单的类型、状态、交易的方向（买卖）、以及一些基础设置等，都是在实现交易机器人功能中常用到的元素，它们都是直接引用自ninjabot库中的model模块。这样做的目的是简化外部对ninjabot内部数据结构和枚举类型的引用，提高代码的可读性和易用性。通过这种方式，开发者可以更直接地访问交易相关的设置、数据帧、序列和订单状态等类型，而无需多次指定完整的包路径。
*/

// 通过类型别名定义的这些结构体名称可以调用model里面相应结构体的方法。
type (
	Settings         = model.Settings         // Settings类型别名，对应交易机器人的基本设置
	TelegramSettings = model.TelegramSettings // TelegramSettings类型别名，用于配置Telegram通知设置
	Dataframe        = model.Dataframe        // Dataframe类型别名，表示市场数据的帧结构
	Series           = model.Series[float64]  // Series类型别名，用于存储一系列浮点数数据
	SideType         = model.SideType         // SideType类型别名，表示订单的买卖方向（买入或卖出）
	OrderType        = model.OrderType        // OrderType类型别名，表示订单的类型
	OrderStatusType  = model.OrderStatusType  // OrderStatusType类型别名，表示订单的状态
)

var (
	SideTypeBuy                    = model.SideTypeBuy                    // 买入类型常量
	SideTypeSell                   = model.SideTypeSell                   // 卖出类型常量
	OrderTypeLimit                 = model.OrderTypeLimit                 // 限价订单类型常量
	OrderTypeMarket                = model.OrderTypeMarket                // 市价订单类型常量
	OrderTypeLimitMaker            = model.OrderTypeLimitMaker            // 只作为挂单方的限价订单类型常量
	OrderTypeStopLoss              = model.OrderTypeStopLoss              // 止损订单类型常量
	OrderTypeStopLossLimit         = model.OrderTypeStopLossLimit         // 限价止损订单类型常量
	OrderTypeTakeProfit            = model.OrderTypeTakeProfit            // 盈利单类型常量
	OrderTypeTakeProfitLimit       = model.OrderTypeTakeProfitLimit       // 限价盈利单类型常量
	OrderStatusTypeNew             = model.OrderStatusTypeNew             // 新订单状态常量
	OrderStatusTypePartiallyFilled = model.OrderStatusTypePartiallyFilled // 部分成交的订单状态常量
	OrderStatusTypeFilled          = model.OrderStatusTypeFilled          // 完全成交的订单状态常量
	OrderStatusTypeCanceled        = model.OrderStatusTypeCanceled        // 取消的订单状态常量
	OrderStatusTypePendingCancel   = model.OrderStatusTypePendingCancel   // 待取消的订单状态常量
	OrderStatusTypeRejected        = model.OrderStatusTypeRejected        // 被拒绝的订单状态常量
	OrderStatusTypeExpired         = model.OrderStatusTypeExpired         // 过期的订单状态常量
)
