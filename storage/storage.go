package storage

import (
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
)

// OrderFilter 是一个函数类型，接受一个 model.Order 对象并返回一个布尔值。
// 它用于根据特定条件判断订单是否满足该条件。
/* 我利用这个过滤器可以找寻符合条件的订单，func WithStatus(status model.OrderStatusType) OrderFilter {
    return func(order model.Order) bool {
        return order.Status == status
    }
}  如状态等于status 的订单 */
type OrderFilter func(model.Order) bool

// Storage 接口定义了操作订单所需的基本方法。
// 包括创建订单、更新订单以及根据过滤器获取订单列表。
type Storage interface {
	CreateOrder(order *model.Order) error                  // 创建订单
	UpdateOrder(order *model.Order) error                  // 更新订单
	Orders(filters ...OrderFilter) ([]*model.Order, error) // 根据一组过滤器获取订单列表
}

// WithStatusIn 返回一个过滤器，该过滤器检查订单的状态是否在指定的状态列表中。
// 如果订单的状态与任何一个给定的状态相匹配，则返回 true。
func WithStatusIn(status ...model.OrderStatusType) OrderFilter {
	return func(order model.Order) bool {
		for _, s := range status {
			if s == order.Status {
				return true
			}
		}
		return false
	}
}

// WithStatus 返回一个过滤器，该过滤器检查订单的状态是否等于指定的状态。
// 如果订单状态与给定状态相等，则返回 true。
func WithStatus(status model.OrderStatusType) OrderFilter {
	return func(order model.Order) bool {
		return order.Status == status
	}
}

// WithPair 返回一个过滤器，该过滤器检查订单的交易对是否等于指定的交易对。
// 如果订单的交易对与给定的交易对相匹配，则返回 true。
func WithPair(pair string) OrderFilter {
	return func(order model.Order) bool {
		return order.Pair == pair
	}
}

// WithUpdateAtBeforeOrEqual 返回一个过滤器，该过滤器检查订单的更新时间是否早于或等于指定的时间。
// 如果订单的更新时间早于或等同于给定时间，则返回 true。
func WithUpdateAtBeforeOrEqual(time time.Time) OrderFilter {
	return func(order model.Order) bool {
		// 如果订单更新时间不晚于指定的时间,就返回true
		return !order.UpdatedAt.After(time)
	}
}
