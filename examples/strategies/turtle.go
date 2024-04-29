package strategies

import (
	"github.com/rodrigo-brito/ninjabot"           // 导入 ninjabot 主库
	"github.com/rodrigo-brito/ninjabot/indicator" // 导入用于计算技术指标的库
	"github.com/rodrigo-brito/ninjabot/service"   // 导入交易服务接口
	"github.com/rodrigo-brito/ninjabot/strategy"  // 导入策略接口
	"github.com/rodrigo-brito/ninjabot/tools/log" // 导入日志工具
)

// 海龟策略结构定义
type Turtle struct{}

// Timeframe 返回策略所使用的时间框架
func (e Turtle) Timeframe() string {
	return "4h" // 使用4小时K线
}

// WarmupPeriod 返回初始化策略所需的历史数据点数量
func (e Turtle) WarmupPeriod() int {
	return 40 // 需要40个数据40个K线数据点进行初始化
}

// Indicators 定义策略需要的技术指标
func (e Turtle) Indicators(df *ninjabot.Dataframe) []strategy.ChartIndicator {
	df.Metadata["max40"] = indicator.Max(df.Close, 40) // 计算40周期内的最高收盘价
	df.Metadata["low20"] = indicator.Min(df.Close, 20) // 计算20周期内的最低收盘价
	return nil
}

// OnCandle 在每个新的K线完成时调用
func (e *Turtle) OnCandle(df *ninjabot.Dataframe, broker service.Broker) {
	closePrice := df.Close.Last(0)          // 获取当前K线的收盘价
	highest := df.Metadata["max40"].Last(0) // 获取过去40个周期的最高价
	lowest := df.Metadata["low20"].Last(0)  // 获取过去20个周期的最低价

	assetPosition, quotePosition, err := broker.Position(df.Pair) // 获取当前交易对的仓位信息
	if err != nil {
		log.Error(err) // 如果查询仓位时出错，则记录错误并返回
		return
	}

	// 如果没有开仓，并且当前价格高于过去40周期的最高价，则开仓买入
	if assetPosition == 0 && closePrice >= highest {
		_, err := broker.CreateOrderMarketQuote(ninjabot.SideTypeBuy, df.Pair, quotePosition/2) // 使用一半的报价货币进行市价买入
		if err != nil {
			log.Error(err) // 如果下单失败，则记录错误
		}
		return
	}

	// 如果已经有仓位，并且当前价格低于过去20周期的最低价，则平仓卖出
	if assetPosition > 0 && closePrice <= lowest {
		_, err := broker.CreateOrderMarket(ninjabot.SideTypeSell, df.Pair, assetPosition) // 市价卖出所有仓位
		if err != nil {
			log.Error(err) // 如果下单失败，则记录错误
		}
	}
}
