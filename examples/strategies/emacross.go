// Package strategies 定义了使用 ninjabot 框架的交易策略。
package strategies

import (
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/indicator"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// CrossEMA 是一个使用指数移动平均线和简单移动平均线的交易策略。
type CrossEMA struct{}

// Timeframe 指定每个蜡烛图的时间间隔。每个持续4小时的蜡烛图数据”指的是4小时K线
func (e CrossEMA) Timeframe() string {
	return "4h" // 4小时蜡烛图间隔。
}

// WarmupPeriod 返回开始策略前需要的过去蜡烛图数量。
func (e CrossEMA) WarmupPeriod() int {
	return 22 // 需要22个蜡烛图就是k线图来进行准确的计算。
}

// Indicators 在数据框架上设置策略使用的指标。
func (e CrossEMA) Indicators(df *ninjabot.Dataframe) []strategy.ChartIndicator {
	df.Metadata["ema8"] = indicator.EMA(df.Close, 8)   // 计算8周期的指数移动平均线。
	df.Metadata["sma21"] = indicator.SMA(df.Close, 21) // 计算21周期的简单移动平均线。

	return []strategy.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "MA's", // 图表上移动平均线的组名。
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["ema8"],
					Name:   "EMA 8",
					Color:  "red",
					Style:  strategy.StyleLine, // EMA8的红色线条样式折线图。
				},
				{
					Values: df.Metadata["sma21"],
					Name:   "SMA 21",
					Color:  "blue",
					Style:  strategy.StyleLine, // SMA21的蓝色线条样式折线图。
				},
			},
		},
	}
}

// 这段代码是 OnCandle 方法的实现，它定义了当每个新的蜡烛图（K线图）完成时，交易策略应如何响应。
func (e *CrossEMA) OnCandle(df *ninjabot.Dataframe, broker service.Broker) {
	closePrice := df.Close.Last(0) // 获取蜡烛图的最新收盘价。
	//调用交易服务的 Position 方法来获取当前交易对（如BTC/USD）的资产和报价货币持仓情况。
	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err)
		return
	}
	// 检查是否有足够的报价货币进行交易如USDT是否大于等于10，且8日的ema是否上涨穿过21日的sma。
	if quotePosition >= 10 &&
		// s相当于df.Metadata["ema8"] ，ref相当于df.Metadata["sma21"]，s数组最后一个数大于ref最后一个数且s倒数第二个数小于或等于ref最后一个数，当我们检测到EMA从下方穿越SMA向上时，可以视为一个买入信号。
		df.Metadata["ema8"].Crossover(df.Metadata["sma21"]) {
		// 计算要买入的资产数量。
		amount := quotePosition / closePrice
		// 下市场订单买入。
		_, err := broker.CreateOrderMarket(ninjabot.SideTypeBuy, df.Pair, amount)
		if err != nil {
			log.Error(err)
		}

		return
	}

	if assetPosition > 0 &&
		// 检查看跌穿越信号（EMA8穿越SMA21以下）。
		df.Metadata["ema8"].Crossunder(df.Metadata["sma21"]) {
		// 下市场订单卖出所有持有。
		_, err = broker.CreateOrderMarket(ninjabot.SideTypeSell, df.Pair, assetPosition)
		if err != nil {
			log.Error(err)
		}
	}
}
