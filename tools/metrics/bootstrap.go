package metrics

import (
	"sort"

	"github.com/samber/lo"    // 导入lo库，用于实现随机抽样
	"gonum.org/v1/gonum/stat" // 导入gonum库，用于统计分析
)

/*
这段Go代码实现了自助法（bootstrap）统计方法，用于从给定数据集中估算统计量的置信区间。代码通过有放回抽样创建多个样本，对每个样本计算一个统计量，然后排序这些统计量以计算均值、标准偏差和置信区间的上下限。这种方法广泛应用于统计分析，特别是在样本量较小或理论分布未知时，为估计结果的可靠性提供了一种强有力的非参数支持。
*/
// BootstrapInterval 结构体定义了一个用于保存置信区间结果的类型，
// 包括下限(Lower)、上限(Upper)、标准偏差(StdDev)和均值(Mean)。

// Mean 均值 ：计算投资组合历史回报数据样本的平均回报率。如果得到的 Mean = 10%，这表明根据历史数据，投资组合的平均预期回报率为10%，如在过去五年里，每年的回报率为：8%, 12%, 10%, 9%, 11%。平均回报率= 50% /5 = 10%
// StdDev 标准偏差：测量投资组合回报率的波动或变异程度。如果 StdDev = 4%，Mean = 10% 这意味着回报率在多数情况下会在10%的基础上上下波动4%，
// Lower (置信区间下界)和 Upper (置信区间下界):表示在给定的置信度下（比如95%），投资组合的真实平均回报率很可能落在这个区间内。如果 Lower = 6% 和 Upper = 14%，这意味着有95%的置信度可以认为，投资组合的回报率在未来一年内会在6%到14%之间。

// 置信度（如 95%）通常是事先设定的，根据分析的目的和所需的准确性来选择，分析的目的和对准确性的需求会影响置信度的选择。例如，在医学或金融领域，由于决策的敏感性和高风险性，通常会选择较高的置信度（如 95% 或 99%）来确保结果的可靠性。
type BootstrapInterval struct {
	Lower  float64
	Upper  float64
	StdDev float64
	Mean   float64
}

// Bootstrap 函数计算一个样本的置信区间，使用的是自助法。
// 参数包括：
// values 是一个浮点数数组，代表一组数据（例如，股票的历史回报率、产品的销售量等），这组数据是你将要进行抽样的基础数据。
// measure 是一个函数，它对每个抽样的子集进行计算并返回一个浮点数。这个函数有可能计算我们关心的统计量（如平均值、中位数、最大值等）
// sampleSize 参数确实指的是样本数量，也就是你从整体数据集中选取的个体数量，用于进行分析或测试。
// confidence 代表置信度，它是一个介于0和1之间的数，表示你想要的置信区间的可靠性。例如，0.95的置信度意味着你希望结果的可靠性达到95%。
func Bootstrap(values []float64, measure func([]float64) float64, sampleSize int,
	confidence float64) BootstrapInterval {
	//初始化了一个空的浮点数切片 data，用于存储每一次抽样得到的统计量结果
	var data []float64

	// 这个 for 循环将重复 sampleSize 次，对应于你希望生成的自助样本的数量。每一次迭代都将生成一个新的样本并计算一个统计量。
	//如这个时原数据[100, 120, 150, 130, 110, 140] 设置sampleSize为1000 意思就从这几个数据中抽取1000次啊 每次抽6个数据然后再平均 。第一次 随机抽取：[150, 120, 100, 120, 150, 140] 平均值：(150 + 120 + 100 + 120 + 150 + 140) / 6 = 130 一直到1000次 ，得到的结果放到一个切片中
	for i := 0; i < sampleSize; i++ {
		//每次抽取的样本都与元数据数量相等如这个时原数据[100, 120, 150, 130, 110, 140]，第一次 随机抽取：[150, 120, 100, 120, 150, 140]
		samples := make([]float64, len(values))
		//如有sampleSize需要抽取的样本1000次，每抽取一次，就要循环len(values)次，意思就是说每抽取一次，得到的样本数量原数据的长度一样
		for j := 0; j < len(values); j++ {
			//这个函数从数组 values 中随机抽取一个元素
			samples[j] = lo.Sample(values)
		}
		//每抽取一次样本measure(samples) 就会被调用一次，它会对 samples 数组中的数据进行计算，根据传过来的参数定义（如平均值、中位数、最大值等）来计算一个具体的统计量。 统计的结果放到切片data切片中
		data = append(data, measure(samples))
	}
	//这个变量计算了在置信区间之外的概率部分。例如，如果confidence是0.95，这意味着你希望95%的情况下你的真实统计量（比如平均值）落在这个区间内。因此，tail将会是0.05，表示有5%的可能性统计量不在这个区间内
	tail := 1 - confidence
	//sort.Float64s(data) 后，data 数组中的元素将会按照从小到大的顺序被重新排列。
	sort.Float64s(data)
	//这里使用stat.MeanStdDev函数计算data中所有数值的平均回报率（mean）和标准偏差（stdDev）
	mean, stdDev := stat.MeanStdDev(data, nil)
	//上界（upper）：计算95%置信区间的上界。1-tail/2计算上界所在的分位数位置。例如，如果tail是0.05，则上界位于97.5%的位置（即1-0.025）
	upper := stat.Quantile(1-tail/2, stat.LinInterp, data, nil)
	//下界（lower）：计算95%置信区间的下界。tail/2计算下界所在的分位数位置。在0.05的tail情况下，下界位于2.5%的位置。
	lower := stat.Quantile(tail/2, stat.LinInterp, data, nil) 

	// 返回包含置信区间下界、上界、标准偏差和均值的结构体
	return BootstrapInterval{
		Lower:  lower,
		Upper:  upper,
		StdDev: stdDev,
		Mean:   mean,
	}
}
