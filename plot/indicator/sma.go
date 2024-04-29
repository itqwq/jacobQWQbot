// 定义指标包，可能用于交易策略或图表绘制中的技术分析。
package indicator

// 导入必要的包。
import (
	"fmt"
	"time"

	"github.com/rodrigo-brito/ninjabot/model" // 可能是用于表示交易数据模型的包。
	"github.com/rodrigo-brito/ninjabot/plot"  // 用于数据可视化的包。

	"github.com/markcheno/go-talib" // 导入TA-Lib，一个技术分析库，提供各种交易指标的计算功能。
)

// SMA 创建一个新的SMA指标实例，接收周期和颜色作为参数。
func SMA(period int, color string) plot.Indicator {
	return &sma{
		Period: period, // 指标计算的周期，例如用过去20个数据点的平均值。。假设每个数据点代表一天的收盘价，那么20日SMA就是最近20天收盘价的算术平均值。
		Color:  color,  // 指标线在图表上显示的颜色。
	}
}

// sma 结构体定义了SMA指标的内部数据结构。
type sma struct {
	Period int                   // 用于计算SMA的时间段长度。如果Period为20，那么就会用过去20个时间单位的数据来计算SMA。
	Color  string                // 指标线的颜色。
	Values model.Series[float64] // 存储计算后的SMA值。 这是一个序列（可能是一个切片或者数组），用来存储经过计算的SMA的数值。这些值是根据Period字段指定的时间段来计算出的。
	Time   []time.Time           // 对应SMA值的时间序列。model.Series[float64]表明这是一个序列，每个元素都是float64类型的数值，表示在特定时间点的SMA值。
}

// Warmup 返回计算指标所需的最小数据点数，即指标的周期(天，小时，分钟等待)。
// 如果你的SMA周期设定为20，则Warmup()方法会返回20。这表示在计算第一个SMA值之前，你需要至少有20个数据点。
func (s sma) Warmup() int {
	return s.Period
}

// Name 返回指标的名称，以格式化的字符串表示，包含其周期。
// fmt.Sprintf("SMA(%d)", s.Period)会创建并返回一个字符串，它以"SMA"开头，后面跟着括号包围的周期数。如果s.Period的值是20，那么Name方法返回的字符串就是"SMA(20)"。这个名字可以在图表显示或者日志记录时用来清楚地指出正在使用或引用的是哪一个特定的SMA指标。
func (s sma) Name() string {
	return fmt.Sprintf("SMA(%d)", s.Period)
}

// Overlay 指示这个指标是否应该在主图表上绘制（与价格图叠加）。
// 这个方法告诉图表绘制软件：将SMA指标画在价格图表上，而不是放在价格图表下方的单独图表中。这样做允许交易者直观地比较价格走势和SMA指标，通常用来判断趋势方向或潜在的买卖点
func (s sma) Overlay() bool {
	return true
}

// Load 从提供的Dataframe中加载数据，并计算SMA值。
// 它使用TA-Lib的Sma函数计算给定周期的SMA。
func (s *sma) Load(dataframe *model.Dataframe) {
	//如果时间序列的数据点数量少于所需的周期，那么就没有足够的数据来计算SMA，所以方法立即返回而不进行进一步的计算。
	if len(dataframe.Time) < s.Period {
		return // 如果提供的数据少于指标周期，则不进行计算。
	}

	// 调用TA-Lib的Sma方法计算SMA，并将结果赋值给s.Values。
	// 计算结果的时间段比输入的时间段短，因为部分数据用于“预热”计算。
	//在计算简单移动平均线（SMA）时，确实需要足够数量的历史数据点来支持计算。因为SMA是通过取特定周期内的数据点的平均值来计算的，如果历史数据点的数量少于这个周期，那么就无法计算出有效的SMA值。如果s.Period为3，且talib.Sma函数返回的切片为[1, 2, 3, 4, 5, 6, 7]，那么表达式[s.Period:]将会创建并返回一个新的切片[4, 5, 6, 7]
	//在实际实现SMA计算时，通常会跳过数据序列的前's.Period' - 1个数据点 因为对于第1个数据点，没有前面的数据来计算平均值。对于第2个数据点（假设周期是10），只有2个数据点，仍然不够计算10日SMA。直到我们达到第10个数据点，这时我们首次有足够的数据（即10个数据点）来计算第一个有效的10日SMA值。
	s.Values = talib.Sma(dataframe.Close, s.Period)[s.Period:]
	//dataframe.Time从索引s.Period开始的子序列赋值给s.Time。通常情况下，s.Period之前的时间戳数据不会被用于计算SMA，因此将其排除在计算结果之外。
	s.Time = dataframe.Time[s.Period:]
}

// Metrics 返回用于图表绘制的指标度量信息。
// 在这个例子中，它返回一个线条样式的度量，包括颜色、值和时间。
func (s sma) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Style:  "line",   // 绘制样式为线条。
			Color:  s.Color,  // 线条颜色。
			Values: s.Values, // 线条对应的值。
			Time:   s.Time,   // 线条对应的时间。
		},
	}
}
