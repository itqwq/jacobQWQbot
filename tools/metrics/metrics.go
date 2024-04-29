package metrics

import (
	"math"

	"gonum.org/v1/gonum/stat"
)

/*
这段代码定义了一个 Go 语言包，用于计算交易指标，包括平均值、支付率和利润因子，这些指标可以帮助评估交易策略的效果和盈利能力，从而为交易决策提供数据支持。
*/
// Mean 函数计算给定 float64 数组中所有元素的平均值，并返回结果。
func Mean(values []float64) float64 {
	return stat.Mean(values, nil) // 使用 gonum 包中的 Mean 函数计算平均值
}

// Payoff 函数计算给定交易结果（如利润或亏损）的支付率。 它会将正收益（胜利）和负收益（失败）分开计算，然后返回胜利与失败的平均比率的绝对值。
// values它是一个包含了交易结果（如利润或亏损）的 float64 数组
func Payoff(values []float64) float64 {
	wins := []float64{}  // 存储正收益（胜利）的切片
	loses := []float64{} // 存储负收益（失败）的切片
	for _, value := range values {
		if value >= 0 {
			wins = append(wins, value) // 如果交易结果为正（收益），将其添加到正收益切片中
		} else {
			loses = append(loses, value) // 如果交易结果为负（亏损），将其添加到负收益切片中
		}
	}

	// 返回胜利与失败的平均比率的绝对值
	//math 包中的 Abs 函数，用于计算参数的绝对值
	//stat.Mean 用于计算平均值
	// 平均比率的绝对值=(把获得正收益的平均值 / 负收益的平均值)取绝对值
	//如果平均比率的绝对值较高，说明胜利交易相对于失败交易的效果更显著，这可能意味着交易策略在盈利方面表现良好。平均比率的绝对值还可以用来评估交易策略的风险。如果绝对值较低，说明胜利和失败交易的效果差异不大，这可能意味着交易策略存在风险不平衡，即在一些交易中获得的收益无法弥补在其他交易中的损失。根据平均比率的绝对值，可以调整交易策略以优化其效果。如果绝对值过低，可以尝试改进风险管理方法或调整交易策略，以增加胜利交易的效果或减少失败交易的影响。
	return math.Abs(stat.Mean(wins, nil) / stat.Mean(loses, nil))
}

// ProfitFactor 函数计算给定交易结果的利润因子。
// 利润因子是指所有盈利交易的总利润与所有亏损交易的总亏损之间的比率。
// 如果不存在亏损交易（即总亏损为零），则返回一个非常大的值10，以避免除以零的错误。
// values 数组是作为参数传递给 ProfitFactor 函数的一个 float64 数组。这个数组包含了交易的结果，每个元素代表一次交易的收益或亏损金额。
func ProfitFactor(values []float64) float64 {
	var (
		wins  float64 // 存储所有盈利交易的总利润
		loses float64 // 存储所有亏损交易的总亏损
	)

	// 遍历交易结果数组，根据正负值将其加入相应的盈利或亏损总额中
	for _, value := range values {
		if value >= 0 {
			wins += value // 如果交易结果为正（收益），将其加入盈利总额中
		} else {
			loses += value // 如果交易结果为负（亏损），将其加入亏损总额中
		}
	}

	// 如果不存在亏损交易（即总亏损为零），则返回一个较大的值10，以避免除以零的错误
	if loses == 0 {
		return 10
	}

	// 返回利润因子，即所有盈利交易的总利润与所有亏损交易的总亏损之间的比率
	//利润因子可以帮助交易者评估他们的交易策略在盈利方面的表现。一个高的利润因子意味着盈利交易的总利润相对于亏损交易的总亏损更大，这表明交易策略在盈利能力方面表现良好。利润因子还可以用来衡量交易策略的风险。如果利润因子较低，即盈利交易的总利润相对于亏损交易的总亏损较小，这可能表示交易策略存在较高的风险。利润因子可以帮助交易者优化他们的交易策略。如果利润因子较低，交易者可以尝试改进交易策略
	return math.Abs(wins / loses)
}
