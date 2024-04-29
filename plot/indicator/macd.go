package indicator

import (
	"fmt"
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"

	"github.com/markcheno/go-talib"
)

// MACD 创建一个移动平均收敛散度指标（MACD）的实例。
// 假设我们有一个股票的价格数据，我们可以计算其MACD指标，并观察MACD线和信号线的交叉点。当MACD线从下方向上穿过信号线时，我们可以考虑买入该股票；而当MACD线从上方向下穿过信号线时，我们可以考虑卖出该股票。
// MACD等于快速EMA减去慢速EMA,MACD线是通过计算快速EMA（12日）和慢速EMA（26日）之间的差值得到的。
// 信号线确实通常是MACD线的9日指数移动平均（EMA），这被视为标准或默认配置。
// 直方图是MACD线与信号线之间的差值。假设某日的MACD值是5，信号线的值是4直方图=5−4=1
func MACD(fast, slow, signal int, colorMACD, colorMACDSignal, colorMACDHist string) plot.Indicator {
	return &macd{
		Fast:            fast,            // 快速EMA的周期。
		Slow:            slow,            // 慢速EMA的周期。
		Signal:          signal,          // MACD信号线的周期。
		ColorMACD:       colorMACD,       // MACD线的颜色。
		ColorMACDSignal: colorMACDSignal, // MACD信号线的颜色。
		ColorMACDHist:   colorMACDHist,   // MACD直方图的颜色。
	}
}

// macd 结构体定义了移动平均收敛散度指标（MACD）的内部数据结构。
type macd struct {
	Fast             int                   // 快速EMA的周期。
	Slow             int                   // 慢速EMA的周期。
	Signal           int                   // MACD信号线的周期。
	ColorMACD        string                // MACD线的颜色。
	ColorMACDSignal  string                // MACD信号线的颜色。
	ColorMACDHist    string                // MACD直方图的颜色。
	ValuesMACD       model.Series[float64] // 存储MACD值。
	ValuesMACDSignal model.Series[float64] // 存储MACD信号线值。
	ValuesMACDHist   model.Series[float64] // 存储MACD直方图值。
	Time             []time.Time           // 对应MACD值的时间序列。
}

// Warmup 返回计算指标所需的最小数据点数，即指标的周期加上信号线周期。
// 快速EMA周期：12天,慢速EMA周期：26天,即26 + 9 = 35。这意味着在计算MACD指标之前，你至少需要35个数据点
func (e macd) Warmup() int {
	return e.Slow + e.Signal
}

// Name 返回指标的名称，以格式化的字符串表示，包含其周期。
func (e macd) Name() string {
	return fmt.Sprintf("MACD(%d, %d, %d)", e.Fast, e.Slow, e.Signal)
}

// Overlay 指示这个指标是否应该在主图表上绘制（与价格图叠加）。
// Overlay方法告诉图表软件或分析工具MACD指标是否应该与价格数据在同一个视图中显示。在这个例子中，通过返回false，它建议将MACD指标放在价格图的下方或旁边的独立区域，以便于分析和解读。
func (e macd) Overlay() bool {
	return false
}

// Load 根据提供的数据加载并计算指标值。
func (e *macd) Load(df *model.Dataframe) {
	//拿慢速周期和信号线相加，因为慢速周期的周期比较长,为了确保慢速EMA的计算精确，需要足够的历史数据。
	warmup := e.Slow + e.Signal
	e.ValuesMACD, e.ValuesMACDSignal, e.ValuesMACDHist = talib.Macd(df.Close, e.Fast, e.Slow, e.Signal)
	//因为预热值是拿来做参考的所以在计算还有分析的时候把他预热值去除掉
	e.Time = df.Time[warmup:]
	e.ValuesMACD = e.ValuesMACD[warmup:]
	e.ValuesMACDSignal = e.ValuesMACDSignal[warmup:]
	e.ValuesMACDHist = e.ValuesMACDHist[warmup:]
}

// Metrics 返回用于图表绘制的指标度量信息。
// 在这个例子中，它返回一个线条样式的度量，包括颜色、值和时间。
func (e macd) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Color:  e.ColorMACD,  // MACD线的颜色。
			Name:   "MACD",       // 指标名称。
			Style:  "line",       // 绘制样式为线条。
			Values: e.ValuesMACD, // MACD线的值。
			Time:   e.Time,       // 对应MACD线值的时间序列。
		},
		{
			Color:  e.ColorMACDSignal,  // MACD信号线的颜色。
			Name:   "MACDSignal",       // 指标名称。
			Style:  "line",             // 绘制样式为线条。
			Values: e.ValuesMACDSignal, // MACD信号线的值。
			Time:   e.Time,             // 对应MACD信号线值的时间序列。
		},
		{
			Color:  e.ColorMACDHist,  // MACD直方图的颜色。
			Name:   "MACDHist",       // 指标名称。
			Style:  "bar",            // 绘制样式为柱状图。
			Values: e.ValuesMACDHist, // MACD直方图的值。
			Time:   e.Time,           // 对应MACD直方图值的时间序列。
		},
	}
}
