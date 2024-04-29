package strategy

// 导入 ninjabot 库的 model 和 service 包
import (
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
)

// Strategy 接口定义了一个交易策略需要实现的基本方法。
type Strategy interface {
	// Timeframe 返回策略执行的时间间隔。例如: "1h", "1d", "1w" 表示每小时、每天、每周。
	Timeframe() string
	// WarmupPeriod 返回执行策略前需要等待的时间，用于为指标加载数据。
	// 这个时间是根据 `Timeframe` 方法指定的周期来衡量的。如果Timeframe()返回"1d"（每天），WarmupPeriod 就表示要求30个 历史1d 的数据 就是拿来参考数据  也可以时20个 历史1d这些历史数据会在策略开始执行之前加载并用于指标的计算和模型的分析。
	WarmupPeriod() int
	// Indicators 对于每个新的K线，都会执行一次，每次调用会传入一个时间帧对应的交易对，然后返回特定交易对指标线列表
	Indicators(df *model.Dataframe) []ChartIndicator
	// OnCandle 对于每个新的K线，在指标被填充后执行，这里可以实现你的交易逻辑。
	// OnCandle 方法在K线关闭后执行。方法的设计就是为了在每个K线周期结束并且该K线数据关闭时被调用。这时，策略会分析这个最新关闭的K线数据以及可能的其他历史数据，以便做出交易决策。
	OnCandle(df *model.Dataframe, broker service.Broker)
}

// HighFrequencyStrategy 接口继承自 Strategy 接口，并添加了处理部分完成K线的方法。
type HighFrequencyStrategy interface {
	Strategy // 继承 Strategy 接口

	// OnPartialCandle 对于每个新的部分完成的K线，在指标被填充后执行。使用OnPartialCandle，策略可以基于最新的市场信息做出快速决策，而不需要等待当前时间段结束。这在需要捕捉短暂市场机会的高频交易策略中特别有用。
	OnPartialCandle(df *model.Dataframe, broker service.Broker)
}
