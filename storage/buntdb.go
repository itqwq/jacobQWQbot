package storage

import (
	"encoding/json"
	"log"
	"strconv"
	"sync/atomic"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/tidwall/buntdb"
)

/* buntdb是一个键值存储数据库，因此它能存储的基本单位是键（key）和值（value）对这意味着你可以存储任何可以被序列化为字符串的数据。文本数据：任何形式的纯文本信息，比如配置项、描述性数据等。

数字：虽然buntdb存储的是字符串，但你可以将数字转换为字符串形式进行存储，使用时再转换回来。

JSON对象：可以将JSON对象序列化为字符串存储，这对于存储结构化数据非常有用。通过buntdb的索引功能，你还可以对存储的JSON数据进行高效的查询和排序操作。

序列化对象：任何可以被序列化的对象，比如使用Go的encoding/gob、encoding/json或其他序列化方式处理过的数据结构，都可以转换为字符串后存储在buntdb中。

二进制数据：虽然buntdb主要处理的是字符串，但你也可以将二进制数据（如图片、文件等）编码为字符串（比如使用Base64编码）来存储。然而，需要注意的是，这种方式可能会增加数据的大小。

列表、地图和其他复杂结构：通过序列化，几乎任何复杂的数据结构都可以转换为字符串形式存储。存取时，只需反序列化即可恢复原始数据结构。 */
// Bunt 结构体包含数据库操作所需的基本属性：lastID 用于追踪最后一个订单的ID，db 是指向 buntdb 数据库实例的指针。
type Bunt struct {
	lastID int64      // 用于订单ID的原子增加 lastID字段确保了数据库中每个订单的唯一标识
	db     *buntdb.DB // buntdb数据库的实例
}

// FromMemory 函数创建一个存储于内存中的buntdb数据库实例，适用于临时存储或测试环境。
// 使用":memory:"作为数据库路径，意味着数据库的所有数据和操作都将仅在内存中进行处理，不会有任何数据被写入磁盘文件。这样的数据库实例在程序运行时创建，在程序结束时销毁，与之相关的所有数据也随之消失。内存数据库非常适合于测试和开发环境，因为你可以在不影响生产数据库的情况下进行操作，并且每次程序重启时数据库都是干净的状态。
func FromMemory() (Storage, error) {
	return newBunt(":memory:")
}

// FromFile 函数允许通过指定文件路径来创建一个持久化的buntdb数据库实例。
func FromFile(file string) (Storage, error) {
	return newBunt(file)
}

// newBunt 是一个辅助函数，根据提供的源文件路径初始化一个Bunt数据库实例。
func newBunt(sourceFile string) (Storage, error) {
	//buntdb.Open(sourceFile): 这个调用尝试打开一个buntdb数据库。sourceFile参数指定了数据库文件的位置。
	db, err := buntdb.Open(sourceFile)
	if err != nil {
		return nil, err // 打开数据库失败
	}

	// 它尝试在数据库上创建一个索引，用于优化基于updated_at字段的查询性能。索引名为"update_index"。
	//"*"是一个索引模式，代表所有的记录都会被索引。
	//buntdb.IndexJSON("updated_at")指定了索引将会基于记录中的updated_at JSON字段来建立。这对于快速检索和排序更新时间的记录非常有用。
	//update_index索引 如同一张特色的表，updated_at 如同一个最后更新时间标记，通过表中的这个标记，可以快速定位和检索到满足这个标记条件的记录。
	//索引在数据库中的作用确实类似于“下标”，它允许数据库系统通过这个下标快速定位到特定的数据库记录。这种机制大大减少了查找记录所需的时间，特别是在处理大量数据时，提高了数据检索的效率。
	err = db.CreateIndex("update_index", "*", buntdb.IndexJSON("updated_at"))
	if err != nil {
		return nil, err // 创建索引失败
	}
	//如果数据库成功打开且索引创建成功,函数创造新的Bunt结构体实例,其`db`字段被设置为刚刚打开的`buntdb.DB`实例
	return &Bunt{
		db: db,
	}, nil
}

// getID 生成一个唯一的ID用于新订单，通过原子增加lastID来保证ID的唯一性和顺序。
// 每次调用getID()方法时，它都会使Bunt实例的lastID值增加一，这个过程是通过原子操作完成的，确保了即使在多个线程或协程并发调用getID()时，每个调用都会得到一个独一无二的、顺序递增的ID。这样就保证了在并发环境下生成的每个ID都是唯一的且不会重复。
// ，直接在内存地址上加一的原因是为了确保在并发访问时操作的原子性和线程安全，这是在多线程环境中生成唯一序列号（如ID）的标准做法。
func (b *Bunt) getID() int64 {
	return atomic.AddInt64(&b.lastID, 1)
}

// CreateOrder 方法接受一个订单对象，将其序列化为JSON后存储到数据库中。每个订单都会被赋予一个唯一ID。
func (b *Bunt) CreateOrder(order *model.Order) error {
	//在这种事务中，除了执行读取操作外，还可以执行插入、更新、删除等修改数据库状态的操作。
	return b.db.Update(func(tx *buntdb.Tx) error {
		order.ID = b.getID() // 分配唯一ID
		content, err := json.Marshal(order)
		if err != nil {
			return err // 订单序列化失败
		}

		_, _, err = tx.Set(strconv.FormatInt(order.ID, 10), string(content), nil)
		return err // 写入数据库
	})
}

// UpdateOrder 方法更新一个已存在的订单。它会将订单对象序列化为JSON并替换数据库中相应的条目。
// 事务控制：通过buntdb.Tx，你可以控制事务的开始、执行操作以及提交或回滚事务。读写操作：在buntdb.Tx事务中，你可以进行数据库的读写操作。隔离级别控制：事务的隔离级别决定了一个事务中的操作在并发环境下如何可见，以及它们如何影响其他事务。
// 插入数据
/* err = db.Update(func(tx *buntdb.Tx) error {
	_, _, err := tx.Set("user:100", `{"name": "John Doe", "age": 30}`, nil)
	return err
}) 先设置tx
 然后使用tx.set 插入数据 */
func (b Bunt) UpdateOrder(order *model.Order) error {
	return b.db.Update(func(tx *buntdb.Tx) error {
		//该方法很可能生成一个唯一的序列号或ID。
		id := strconv.FormatInt(order.ID, 10)
		//使用json.Marshal函数将order对象序结构体列化为JSON格式。如{"id":1,"name":"Apple Watch","total":299.99}

		content, err := json.Marshal(order)
		if err != nil {
			return err // 订单序列化失败
		}
		//。在这个上下文中，id 作为键（key），而将 content（已被转换成字符串的形式）作为值（value）存储到BuntDB数据库中。
		_, _, err = tx.Set(id, string(content), nil)
		return err // 更新数据库条目
	})
}

// Orders 方法根据提供的过滤器函数检索并返回符合条件的订单列表。
func (b Bunt) Orders(filters ...OrderFilter) ([]*model.Order, error) {
	orders := make([]*model.Order, 0)
	//b.db.View启动的是一个只读事务
	err := b.db.View(func(tx *buntdb.Tx) error {
		//遍历关于update_index 索引顺序遍历数据库的记录 ，key是记录的唯一标识符，而value是存储的数据，返回值: bool是这个函数的返回类型，用于控制遍历的流程。在BuntDB中，如果这个回调函数返回true，则遍历继续；如果返回false，则遍历停止。
		//意思就是我执行方法时自动提供的。当你调用tx.Ascend方法并传递一个索引名（在这个例子中是"update_index"）与一个回调函数时，BuntDB会自动遍历该索引下的所有记录。对于每一条记录，BuntDB将其键（key）和值（value）作为参数传递给你提供的回调函数。
		err := tx.Ascend("update_index", func(key, value string) bool {
			var order model.Order
			//json.Unmarshal([]byte(value), &order): 这行代码将记录的值（value），即JSON字符串，反序列化成model.Order类型的对象。
			//它将value字符串（JSON格式）转换回Go语言的结构体实例。 json.Unmarshal函数然后解析这个字节切片，根据JSON数据中的键和结构体中的字段标签（tags）匹配，将相应的值填充到order指向的结构体实例中。
			// value值本来是json后面转成字节切片，因为json.Unmarshal需要的输入是一个字节切片（[]byte），这是因为在Go语言中，底层对字符串和字节序列的处理是通过字节切片实现的。
			err := json.Unmarshal([]byte(value), &order)
			if err != nil {
				log.Println(err)
				//在tx.Ascend方法的上下文中，回调函数返回true意味着“继续遍历其他记录”，而返回false则意味着“停止遍历”。
				return true // 继续遍历
			}
			//遍历所有过滤器
			for _, filter := range filters {
				//这行代码的作用是检查当前订单是否满足某个特定的过滤条件。如果不满足（即过滤器函数返回false），则通过return true跳过当前订单，继续遍历数据库中的下一条记录。
				if ok := filter(order); !ok {
					return true // 当前订单不符合过滤条件，继续遍历
				}
			}
			//把符合条件的&order放到orders
			orders = append(orders, &order)

			return true // 继续遍历直到结束
		})
		return err // 返回遍历过程中可能出现的错误
	})
	if err != nil {
		return nil, err // 如果在查看数据库时发生错误，返回错误
	}
	return orders, nil // 返回满足条件的订单列表
}
