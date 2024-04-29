package strategy

// 导入的包有 time 和 ninjabot 的 model 包。
import (
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
)

// MetricStyle 定义了一个字符串类型，用来表示图表指标的不同显示风格。比如，折线图 柱形图等
type MetricStyle string

// 下面是 MetricStyle 可能的值，用于指定指标在图表中的呈现方式。
const (
	StyleBar       = "bar"       // 柱状图：用于表示变量的大小，常用于比较不同项目。
	StyleScatter   = "scatter"   // 散点图：用于描绘变量之间的关系。
	StyleLine      = "line"      // 折线图：用于显示数据随时间变化的趋势。
	StyleHistogram = "histogram" // 直方图：用于显示数据的分布情况。
	StyleWaterfall = "waterfall" // 瀑布图：用于显示一个序列数据的累积效果。
)

// IndicatorMetric 结构体定义了一个用于图表的指标度量。
type IndicatorMetric struct {
	Name   string                // 指标的名称。
	Color  string                // 指标在图表中的颜色。
	Style  MetricStyle           // 指标的显示风格，默认为折线图。
	Values model.Series[float64] // 指标的值序列，采用泛数类型。是用来存储指标的具体数值序列的,如果是移动平均线指标，那么 Values 中存储的就是每个时间点上的移动平均线的数值。
}

// ChartIndicator 结构体定义了用于图表的指标数据结构。
// ChartIndicator像是一个大容器，它可以包含多个IndicatorMetric（小盒子，每个盒子都包含了一条特定数据线的详细信息。例如，你想展示一个8日指数移动平均线（EMA 8），你就可以创建一个IndicatorMetric实例，设置它的名字为"EMA 8"，颜色为红色，样式为直线，然后把这条线的数据值放进去。），可以在这个大容器上设置一些属性，如时间序列，示指标是否应该被覆盖在价格图表上。指标的分组名称，用于图表中进行逻辑分组。需要预热的数据点数，指标计算之前所需的最小数据点
type ChartIndicator struct {
	Time      []time.Time       // 时间序列，指标数据对应的时间点。
	Metrics   []IndicatorMetric // 一组指标度量，每个度量包括名称、颜色、风格和值。
	Overlay   bool              // 指示指标是否应该被覆盖在价格图表上。
	GroupName string            // 指标的分组名称，用于图表中进行逻辑分组。
	Warmup    int               // 需要预热的数据点数，指标计算之前所需的最小数据点。
}
