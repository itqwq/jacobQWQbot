package indicator

// 引入必要的包
import (
	"fmt"  // 用于格式化字符串
	"time" // 用于处理时间

	"github.com/rodrigo-brito/ninjabot/model" // ninjabot库中的模型，用于交易数据处理
	"github.com/rodrigo-brito/ninjabot/plot"  // ninjabot库的绘图部分，用于指标的图表展示

	"github.com/markcheno/go-talib" // TA-Lib库，用于计算各种技术指标
)

// Stoch函数定义了一个用于生成随机振荡器指标的函数
// 超买区域：当随机振荡器的值高于80时，表明市场可能处于超买状态。在这种情况下，价格可能面临回调或反转的风险，因为它被认为是高估的。
// 超卖区域：当随机振荡器的值低于20时，表明市场可能处于超卖状态。这意味着价格可能即将反弹，因为它被认为是低估的。
func Stoch(fastK, slowK, slowD int, colorK, colorD string) plot.Indicator {
	// 返回一个stoch结构体的实例，配置了相关参数
	return &stoch{
		FastK:  fastK,  // 快速K线周期
		SlowK:  slowK,  // 慢速K线平滑周期
		SlowD:  slowD,  // 慢速D线平滑周期
		ColorK: colorK, // K线颜色
		ColorD: colorD, // D线颜色
	}
}

// stoch结构体保存随机振荡器指标的设置和计算结果
type stoch struct {
	FastK   int                   // 快速K线周期
	SlowK   int                   // 慢速K线平滑周期
	SlowD   int                   // 慢速D线平滑周期
	ColorK  string                // K线颜色
	ColorD  string                // D线颜色
	ValuesK model.Series[float64] // 计算后的K线值
	ValuesD model.Series[float64] // 计算后的D线值
	Time    []time.Time           // 对应的时间序列
}

// Warmup方法返回需要预热的数据长度，以确保指标计算的可靠性
func (e stoch) Warmup() int {
	return e.SlowD + e.SlowK // 预热期取决于慢速K线和慢速D线的周期
}

// Name方法返回指标的名称和配置的周期参数
func (e stoch) Name() string {
	return fmt.Sprintf("STOCH(%d, %d, %d)", e.FastK, e.SlowK, e.SlowD)
}

// Overlay方法指示该指标不需要绘制在主价格图上
func (e stoch) Overlay() bool {
	return false
}

// Load方法负责加载数据帧并使用TA-Lib计算随机振荡器的K线和D线值
// e.FastK：快速%K线的周期数。e.SlowK：慢速%K线的平滑周期数。e.SlowD：慢速%D线的平滑周期数。talib.SMA：用于慢速%D线的平滑方法，这里同样使用简单移动平均。
func (e *stoch) Load(dataframe *model.Dataframe) {
	e.ValuesK, e.ValuesD = talib.Stoch(
		dataframe.High, dataframe.Low, dataframe.Close,
		e.FastK, e.SlowK, talib.SMA, e.SlowD, talib.SMA, // 使用SMA作为平滑函数
	)
	e.Time = dataframe.Time // 时间序列直接采用数据帧中的时间
}

// Metrics方法定义了如何在图表上展示计算出的K线和D线值
func (e stoch) Metrics() []plot.IndicatorMetric {
	// 返回K线和D线的图表配置，包括颜色和样式
	return []plot.IndicatorMetric{
		{
			Color:  e.ColorK, // K线颜色
			Name:   "K",
			Style:  "line",
			Values: e.ValuesK,
			Time:   e.Time,
		},
		{
			Color:  e.ColorD, // D线颜色
			Name:   "D",
			Style:  "line",
			Values: e.ValuesD,
			Time:   e.Time,
		},
	}
}
