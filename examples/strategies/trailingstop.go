package strategies

import (
	// 导入所需的包，以使用 ninjabot 交易机器人
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/indicator" // 指标计算
	"github.com/rodrigo-brito/ninjabot/model"     // 数据模型
	"github.com/rodrigo-brito/ninjabot/service"   // 交易服务
	"github.com/rodrigo-brito/ninjabot/strategy"  // 交易策略
	"github.com/rodrigo-brito/ninjabot/tools"     // 工具集
	"github.com/rodrigo-brito/ninjabot/tools/log" // 日志记录工具
)

// trailing 结构定义了一个使用跟踪止损的交易策略，适用于高频交易环境。
type trailing struct {
	// 使用 map 存储每个交易对的跟踪止损对象，跟踪止损可以在实时调整止损价格以保护利润并减少损失。
	trailingStop map[string]*tools.TrailingStop
	// 存储每个交易对的调度器对象，调度器用于管理定时任务。
	scheduler map[string]*tools.Scheduler
}

// NewTrailing 创建一个新的跟踪策略实例。
func NewTrailing(pairs []string) strategy.HighFrequencyStrategy {
	strategy := &trailing{
		trailingStop: make(map[string]*tools.TrailingStop), // 初始化跟踪止损对象的映射
		scheduler:    make(map[string]*tools.Scheduler),    // 初始化调度器对象的映射
	}

	// 为每个交易对初始化跟踪止损和调度器
	for _, pair := range pairs {
		strategy.trailingStop[pair] = tools.NewTrailingStop() // 创建新的跟踪止损对象
		strategy.scheduler[pair] = tools.NewScheduler(pair)   // 创建新的调度器
	}

	return strategy
}

// Timeframe 返回策略使用的时间框架，这里是4小时。
func (t trailing) Timeframe() string {
	return "4h"
}

// WarmupPeriod 返回初始化策略所需的历史数据点数量，这里需要21个数据点。
func (t trailing) WarmupPeriod() int {
	return 21
}

// Indicators 定义策略所需的技术指标，这里计算了快速EMA和慢速SMA。
func (t trailing) Indicators(df *model.Dataframe) []strategy.ChartIndicator {
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, 8)  // 计算8周期的指数移动平均
	df.Metadata["sma_slow"] = indicator.SMA(df.Close, 21) // 计算21周期的简单移动平均
	return nil                                            // 无需返回可视化指标
}

// OnCandle 每次新的蜡烛图完成时调用。
func (t trailing) OnCandle(df *model.Dataframe, broker service.Broker) {
	// 获取当前交易对的资产和报价货币的仓位
	asset, quote, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err) // 如果查询仓位时出错，则记录错误并返回
		return
	}

	// 如果有足够的现金，并且没有持仓，且快速EMA穿越慢速SMA向上
	// 条件包括：报价货币余额大于10，资产的市值小于10，且快速EMA（8周期）刚好穿过慢速SMA（21周期）向上
	if quote > 10.0 && asset*df.Close.Last(0) < 10 && df.Metadata["ema_fast"].Crossover(df.Metadata["sma_slow"]) {
		// 创建市价订单买入，并使用当前报价货币数量
		// 执行市场买单，使用全部可用的报价货币
		_, err = broker.CreateOrderMarketQuote(ninjabot.SideTypeBuy, df.Pair, quote)
		if err != nil {
			log.Error(err) // 如果创建订单失败，则记录错误并返回
			return
		}

		// 启动跟踪止损，通过这个方法设置最新价格，还有止损价格
		// 跟踪止损用于动态调整止损价，从而在价格回调时保护利润或减少损失
		//例如：周期开始时，比特币的收盘价是 $10,000。该周期的最低价是 $9,800。意味着设置当前市场价格为 $10,000，同时将止损价格设为 $9,800。如果在下一个周期内，比特币的价格最高达到了 $10,500，而该周期的最低价是 $10,200。此时，您可能再次评估并调用 t.trailingStop[df.Pair].Start(10500, 10200)，更新止损价格到 $10,200。 一个周期就是1h、1d等k线值
		t.trailingStop[df.Pair].Start(df.Close.Last(0), df.Low.Last(0))
		return
	}
}

// OnPartialCandle 在部分蜡烛图完成时调用。
func (t trailing) OnPartialCandle(df *model.Dataframe, broker service.Broker) {
	// 这段代码的作用是检查对应交易对的跟踪止损对象是否存在，并询问该对象基于当前的市场价格是否应该触发止损。调用Update更新功能，传入当前的最新价格。这个 Update 方法负责检查当前价格是否已达到或跌破止损价格，如果是，将返回 true 就执行if里面的操作
	if trailing := t.trailingStop[df.Pair]; trailing != nil && trailing.Update(df.Close.Last(0)) {
		// 获取当前资产仓位
		asset, _, err := broker.Position(df.Pair)
		if err != nil {
			log.Error(err)
			return
		}

		// 如果 trailing.Update() 返回 true（意味着触发了止损），并且当前还持有资产（asset > 0），则执行卖出操作。
		if asset > 0 {
			// 以市价创建卖出订单，卖出全部持仓
			_, err = broker.CreateOrderMarket(ninjabot.SideTypeSell, df.Pair, asset)
			if err != nil {
				log.Error(err)
				return
			}
			// 停止跟踪止损
			trailing.Stop()
		}
	}
}
