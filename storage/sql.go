package storage

import (
	"time"

	"github.com/samber/lo" // 引入lo包用于函数式操作切片，如过滤
	"gorm.io/gorm"         // GORM ORM库，用于操作SQL数据库

	"github.com/rodrigo-brito/ninjabot/model" // 自定义的model包，定义了Order等模型
)

// SQL结构体，封装了对数据库操作的方法
type SQL struct {
	db *gorm.DB // gorm.DB对象，用于数据库操作
}

// FromSQL 函数初始化一个SQL数据库连接，返回一个SQL存储对象
// dialect: 这是一个gorm.Dialector接口的实例，它定义了GORM如何与特定的数据库类型进行交云。你可以传入任何GORM支持的数据库方言，比如MySQL、PostgreSQL等。
// opts...表示FromSQL函数可以接受零个或多个gorm.Option类型的参数，这些参数用于自定义GORM的行为。
// gorm.Option是一个接口类型，用于传递配置选项给GORM的初始化函数gorm.Open，以定制GORM的行为。这些选项可以包括日志配置、数据库连接池设置、命名策略等。
func FromSQL(dialect gorm.Dialector, opts ...gorm.Option) (Storage, error) {
	// 使用gorm.Open连接数据库
	//db：函数的第一个返回值是*gorm.DB类型的对象。 封装了数据库操作的方法，比如查询、插入、更新和删除。通过这个对象，你可以进行数据库交云，执行SQL命令。
	db, err := gorm.Open(dialect, opts...)
	if err != nil {
		return nil, err // 连接失败，返回错误
	}
	//db.DB()充当了桥接器的角色。利用GORM的便利性进行日常的数据库操作。创建（Create）读取（Read）,更新（Update）,删除（Delete）,关联操作（Associations）,原生SQL和SQL构建器（Raw SQL and SQL Builder）,迁移（Migrations）
	sqlDB, err := db.DB() // 从*gorm.DB对象中获取底层的*sql.DB对象
	if err != nil {
		return nil, err
	}

	// 设置数据库连接池参数
	sqlDB.SetMaxIdleConns(10)           // 设置空闲连接池中连接的最大数量，空闲连接就是那些当前没有被用于数据操作的连接。只能有这么多空余
	sqlDB.SetMaxOpenConns(100)          // 设置打开数据库连接的最大数量，限制帮助你确保数据库不会因为尝试处理太多请求而过载，类似于确保你的餐厅不会因为接待过多的顾客而服务质量下降或资源紧张。
	sqlDB.SetConnMaxLifetime(time.Hour) // 设置了连接可复用的最大时间，意味着每个数据库连接只能连续工作设定的时长。一旦连接使用时间达到这个设定值，即便这个连接还能继续使用，系统也会关闭它，并在需要时创建新的连接。

	// 自动迁移数据库，创建或更新数据库表结构
	err = db.AutoMigrate(&model.Order{})
	if err != nil {
		return nil, err // 迁移失败，返回错误
	}

	// 返回SQL存储对象 ，虽然声明的返回类型是 Storage 接口，但实际上返回的是 SQL 结构体的指针，而 SQL 结构体实现了 Storage 接口。这种情况下，因为 SQL 类型实现了 Storage 接口的所有方法，所以可以将 *SQL 类型的指针作为 Storage 接口类型的值返回，而不会引发编译错误。
	return &SQL{
		db: db,
	}, nil
}

// CreateOrder 方法在数据库中创建一个新的订单记录
func (s *SQL) CreateOrder(order *model.Order) error {
	result := s.db.Create(order) // 使用GORM的Create方法添加记录
	return result.Error          // 返回可能出现的错误
}

// UpdateOrder 方法更新一个已存在的订单记录
func (s *SQL) UpdateOrder(order *model.Order) error {
	o := model.Order{ID: order.ID} // 根据ID查找订单，首先创建一个 model.Order 类型的变量 o，并设置其ID为传入的订单对象 order 的ID。这样做是为了根据订单的ID查找对应的订单记录。
	s.db.First(&o)                 // 查找第一个匹配的订单
	o = *order                     // 使用传入的订单对象 order 中的数据更新变量 o 中的数据。
	result := s.db.Save(&o)        // 保存更改 ，调用 Save 方法将更新后的订单记录保存到数据库中。这里传入了 &o，即指向变量 o 的指针，以确保在保存后可以更新原始的订单对象。
	return result.Error            // 返回可能出现的错误
}

// Orders 方法根据提供的过滤器函数检索并返回符合条件的订单列表
func (s *SQL) Orders(filters ...OrderFilter) ([]*model.Order, error) {
	orders := make([]*model.Order, 0) // 初始化订单切片
	//在这种情况下，Find 方法的行为类似于 SQL 中的 SELECT * FROM table，它会检索数据库中的所有记录。即使没有显式提供任何条件，它仍然会返回数据库中的所有记录。
	result := s.db.Find(&orders) // 使用GORM的Find方法查找所有订单，查找条件是空的，该语句将返回数据库中的所有订单记录，并将它们存储在 orders 切片中。
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return orders, nil // 如果出错且错误不是未找到记录，返回空切片和nil，这种情况下，nil 错误表示操作没有失败，而是未找到任何记录，因此不需要进一步处理。
	}

	// 使用lo包的Filter函数过滤订单，返回符合所有过滤条件的订单列表
	//lo.Filter该函数接受两个参数：要过滤的切片（这里是订单列表 orders）和一个函数。这个函数用于定义过滤的条件，它接受切片元素和索引作为参数，并返回一个布尔值，表示是否保留该元素。
	//每个过滤器都是一个函数，每个函数都是之前我们定义的条件，用于检查订单是否满足特定的条件
	return lo.Filter(orders, func(order *model.Order, _ int) bool {
		//for _, filter := range filters {...}：遍历传入的过滤器列表 filters。每个过滤器是一个函数，用于检查订单是否符合特定的条件。
		for _, filter := range filters {
			//if !filter(*order) {...}：对于每个订单，依次应用所有的过滤器。如果某个过滤器返回 false，表示该订单不符合过滤条件，那么立即返回 false，表示该订单应该被排除。
			if !filter(*order) {
				return false
			}
		}
		//如果订单通过了所有的过滤器，那么返回 true，表示该订单符合所有的过滤条件，应该保留。
		return true
	}), nil
}
