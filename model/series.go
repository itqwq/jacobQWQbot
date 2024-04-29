// 定义model包
package model

import (
	"strconv" // 引入strconv包，用于字符串和基本类型的转换
	"strings" // 引入strings包，提供用于操作字符串的函数

	"golang.org/x/exp/constraints" // 引入constraints包，提供泛型约束
)

// Series是一个时间序列数据的泛型切片
// ype Series[T constraints.Ordered] []T  意思可以定义 任何符合 constraints.Ordered约束类型，约束类型有可以使用比较运算符 <、>、<= 和 >= 进行比较的类型，就是包括整数，浮点，字符串类型。 T constraints.Ordered] 这是放数据类型的地方的， []T这是放数据的地方如Series[int]{10, 12, 9, 7, 13, 15, 8}
type Series[T constraints.Ordered] []T

// Values返回序列中的所有值
func (s Series[T]) Values() []T {
	return s // 直接返回序列s
}

// Lenght返回序列中值的数量
func (s Series[T]) Lenght() int {
	return len(s) // 返回序列s的长度
}

// Last返回序列中给定过去索引位置的最后一个值
func (s Series[T]) Last(position int) T {
	return s[len(s)-1-position] // 根据给定的position计算并返回相应位置的值
}

// LastValues返回序列中给定大小的最后几个值
func (s Series[T]) LastValues(size int) []T {
	if l := len(s); l > size {
		return s[l-size:] // 如果序列长度大于请求的size，则返回序列的最后size个值
	}
	return s // 否则返回整个序列
}

// Crossover判断序列的最后一个值是否大于参考序列的最后一个值 如果符合则买入
func (s Series[T]) Crossover(ref Series[T]) bool {
	// 如果s的最后一个值大于ref的最后一个值，并且s的倒数第二个值不大于ref的倒数第二个值，则返回true
	return s.Last(0) > ref.Last(0) && s.Last(1) <= ref.Last(1)
}

// Crossunder判断序列的最后一个值是否小于参考序列的最后一个值 卖出信号
func (s Series[T]) Crossunder(ref Series[T]) bool {
	// 如果s的最后一个值小于或等于ref的最后一个值，并且s的倒数第二个值大于ref的倒数第二个值，则返回true
	return s.Last(0) <= ref.Last(0) && s.Last(1) > ref.Last(1)
}

// Cross判断序列的最后一个值是否与参考序列的最后一个值存在交叉
func (s Series[T]) Cross(ref Series[T]) bool {
	// 如果s的最后一个值大于ref的最后一个值或小于ref的最后一个值，则返回true
	return s.Crossover(ref) || s.Crossunder(ref)
}

// NumDecPlaces返回一个float64值的小数位数
func NumDecPlaces(v float64) int64 {
	s := strconv.FormatFloat(v, 'f', -1, 64) // 将float64值v转换为字符串s
	i := strings.IndexByte(s, '.')           // 查找s中'.'的索引位置i
	if i > -1 {                              // 如果找到'.'，则计算并返回小数位数
		return int64(len(s) - i - 1)
	}
	return 0 // 如果没有找到'.'，则返回0
}
