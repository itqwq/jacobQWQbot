package tools

import (
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

/*
这段代码定义了一个基于Go语言的加密货币交易调度系统，使用ninjabot库构建。它包含自定义买卖条件的功能，通过设置的条件自动执行市场订单。主要包括两种类型的交易操作：买入和卖出，每种操作都通过特定的条件函数触发。调度器通过监测市场数据，根据这些条件动态管理和执行交易订单，旨在减少手动操作的需要，实现交易策略的自动化。
*/
// OrderCondition 结构体用于定义订单交易条件
type OrderCondition struct {
	//该函数的目的是基于提供的市场数据帧 df（可能包含价格、成交量等信息）来决定是否满足某个特定的交易条件。如果返回 true，则意味着满足交易条件；如果返回 false，则不满足。
	Condition func(df *ninjabot.Dataframe) bool // Condition 是一个函数，决定是否触发交易
	Size      float64                           // Size 表示交易的数量
	Side      ninjabot.SideType                 // Side 表示交易方向（买入或卖出）
}

// Scheduler 结构体用于管理交易调度
type Scheduler struct {
	pair            string           // pair 表示交易对，例如 "BTC/USD"
	orderConditions []OrderCondition // orderConditions 存储所有交易条件
}

// NewScheduler 创建一个新的Scheduler实例
// &拿到一个地址，然后*就是把地址解析成值，两个配合使用
func NewScheduler(pair string) *Scheduler {
	return &Scheduler{pair: pair} // 初始化一个Scheduler，设置其交易对
}

// SellWhen 添加一个卖出条件
func (s *Scheduler) SellWhen(size float64, condition func(df *ninjabot.Dataframe) bool) {
	//将新元素添加到切片（slice）的末尾。在这里，它被用来将一个新的 OrderCondition 添加到 Scheduler 的 orderConditions 切片中
	s.orderConditions = append(
		s.orderConditions,
		OrderCondition{Condition: condition, Size: size, Side: ninjabot.SideTypeSell},
	) // 将一个新的卖出条件添加到订单条件列表
}

// 添加一个买入条件
func (s *Scheduler) BuyWhen(size float64, condition func(df *ninjabot.Dataframe) bool) {
	s.orderConditions = append(
		s.orderConditions,
		OrderCondition{Condition: condition, Size: size, Side: ninjabot.SideTypeBuy},
	)
}

// Update 方法根据当前市场数据更新交易条件，并尝试执行满足条件的交易。
func (s *Scheduler) Update(df *ninjabot.Dataframe, broker service.Broker) {
	// 使用 lo.Filter 来过滤和更新 orderConditions 切片。
	// lo.Filter 返回一个新的切片，其中只包含满足指定条件的元素。
	// 那这段代码意思就是， 告诉lo.Filter函数，过滤的主体是一个[OrderCondition]切片，然后在s.orderConditions过滤，因为s.orderConditions包含着多个[OrderCondition]切片，过滤的条件是func(oc OrderCondition, _ int)函数，如果 func 函数返回 true，则该 OrderCondition 元素会被保留在新的返回切片中，如果 func 函数返回 false，则该 OrderCondition 元素不会被保留在新的返回切片中
	s.orderConditions = lo.Filter[OrderCondition](s.orderConditions, func(oc OrderCondition, _ int) bool {
		// 检查当前交易时间帧市场数据是否满足某个交易条件。
		//如果交易条件满足（即 oc.Condition(df) 返回 true），则尝试创建市场订单如果订单创建时出现错误（err != nil），则记录错误并返回 true。这些数据没有被从切片中移除，还是留着原来的切片中，如果订单创建成功，返回 false。当满足条件的订单被创建成功时，它将从 s.orderConditions 切片中移除，不再留在其中。这个过滤操作会返回一个新的切片，其中包含了原始切片中符合条件的元素。
		if oc.Condition(df) {
			// 如果条件满足，尝试通过交易经纪创建市场订单。
			_, err := broker.CreateOrderMarket(oc.Side, s.pair, oc.Size)
			if err != nil {
				// 如果创建订单时发生错误，记录错误并返回 true，以保留此条件以便后续重试。
				log.Error(err)
				return true
			}
			// 如果订单创建成功，返回 false，从切片中移除此条件。
			return false
		}
		// 如果当前市场数据不满足条件，返回 true，以保留此条件。
		return true
	})
}
