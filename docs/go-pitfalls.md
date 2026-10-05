# Go 进阶避坑指南

## 写在前面

Go 语法简单，上手快，但"能跑"和"跑得对"之间有不少坑。这些坑大多不会报编译错误，运行时才炸，而且往往是偶发的、难复现的。

本文整理了日常开发中最容易踩的 10 类问题，每个坑都给出：**错误写法 → 为什么错 → 正确写法**。适合有一定 Go 基础、写过一些业务代码的同学。

---

## 一、Slice 篇

### 1.1 append 可能改了别人的数据

```go
a := make([]int, 3, 5) // len=3, cap=5
a[0], a[1], a[2] = 1, 2, 3

b := append(a, 4) // b 和 a 共享底层数组
b[0] = 999        // a[0] 也变成了 999！
```

**为什么：** slice 是引用类型，append 在容量够用时不会创建新数组，b 和 a 指向同一块内存。改 b 等于改 a。

**正确做法：** 如果不想互相影响，先 copy 一份：

```go
b := make([]int, len(a))
copy(b, a)
b = append(b, 4)
```

### 1.2 子切片写穿

```go
data := []int{1, 2, 3, 4, 5}
sub := data[1:3] // sub = [2, 3]，但 cap 还很大

sub = append(sub, 999) // 没有超出 data 的容量
// data 变成了 [1, 2, 3, 999, 5]，第4个元素被覆盖了！
```

**为什么：** 子切片和原切片共享底层数组。append 没超容量时直接写，写到了原数组的位置上。

**正确做法：** 用三下标切片限制容量：

```go
sub := data[1:3:3] // 第三个数字限制 cap = 3-1 = 2
sub = append(sub, 999) // cap 不够了，会新建数组，不影响 data
```

### 1.3 nil slice 和空 slice 的区别

```go
var a []int          // nil slice, len=0, cap=0
b := []int{}         // 空 slice, len=0, cap=0
c := make([]int, 0)  // 空 slice, len=0, cap=0

fmt.Println(a == nil) // true
fmt.Println(b == nil) // false
```

**影响：** 序列化成 JSON 时，nil slice 变成 `null`，空 slice 变成 `[]`。给前端返数据如果不注意，前端拿到 null 可能报错。

**正确做法：** 给前端返列表时，确保初始化为空 slice 而不是 nil：

```go
result := make([]Item, 0) // 而不是 var result []Item
```

---

## 二、Map 篇

### 2.1 并发读写 map 直接 panic

```go
m := make(map[string]int)

// goroutine A 写
go func() {
    for { m["key"] = 1 }
}()

// goroutine B 读
go func() {
    for { _ = m["key"] }
}()

// 程序直接 panic: concurrent map read and map write
```

**为什么：** Go 的 map 不是并发安全的。两个 goroutine 同时读写会触发运行时检测，直接崩溃，不是数据错乱而是直接挂。

**正确做法：** 用 `sync.RWMutex` 或 `sync.Map`：

```go
// 方案一：加锁
var mu sync.RWMutex
mu.Lock()
m["key"] = 1
mu.Unlock()

mu.RLock()
_ = m["key"]
mu.RUnlock()

// 方案二：用 sync.Map（读多写少场景更合适）
var sm sync.Map
sm.Store("key", 1)
val, ok := sm.Load("key")
```

### 2.2 往 nil map 写数据直接 panic

```go
var m map[string]int
m["key"] = 1 // panic: assignment to entry in nil map
```

**为什么：** `var m map[...]` 声明后 m 是 nil，读 nil map 不报错（返回零值），但写 nil map 直接 panic。

**正确做法：** 用之前先 make：

```go
m := make(map[string]int)
m["key"] = 1
```

---

## 三、Goroutine 篇

### 3.1 循环里启动 goroutine，变量全是最后一个

```go
names := []string{"张三", "李四", "王五"}

for _, name := range names {
    go func() {
        fmt.Println(name) // 大概率打印三次"王五"
    }()
}
```

**为什么：** 闭包捕获的是变量 `name` 的引用，不是值。循环跑得比 goroutine 启动快，等 goroutine 真正执行时，name 已经变成最后一个值了。

**正确做法：** 把变量作为参数传进去：

```go
for _, name := range names {
    go func(n string) {
        fmt.Println(n) // 正确：张三、李四、王五
    }(name)
}
```

> Go 1.22+ 已经修复了这个问题（循环变量改为每次迭代新建），但很多项目还在用老版本，养成传参的习惯更稳。

### 3.2 goroutine 泄漏

```go
func fetch() <-chan string {
    ch := make(chan string) // 无缓冲 channel
    go func() {
        result := doSomething()
        ch <- result // 如果没人读 ch，这行永远阻塞
    }()
    return ch
}

// 调用方
ch := fetch()
// 如果因为超时等原因没有 <-ch，那个 goroutine 永远卡在那里，内存泄漏
```

**为什么：** 往无缓冲 channel 写数据时，如果没有接收方，发送方会永远阻塞。goroutine 不会被 GC 回收，越积越多。

**正确做法：** 用带缓冲的 channel，或者用 context 控制超时：

```go
ch := make(chan string, 1) // 缓冲 1，写完就走，不卡
```

---

## 四、Defer 篇

### 4.1 循环里用 defer，资源不会及时释放

```go
for _, file := range files {
    f, err := os.Open(file)
    if err != nil { continue }
    defer f.Close() // 不会在每次循环结束时执行！
    // 而是等整个函数 return 时才全部执行
}
// 如果 files 有 10000 个，会同时打开 10000 个文件句柄
```

**为什么：** defer 是在**函数返回时**执行，不是在代码块结束时。循环不是函数，所以 defer 会堆积。

**正确做法：** 把循环体包成独立函数，或者手动 Close：

```go
// 方案一：包成函数
for _, file := range files {
    func() {
        f, _ := os.Open(file)
        defer f.Close() // 匿名函数返回时就执行
        // 处理文件
    }()
}

// 方案二：手动关
for _, file := range files {
    f, _ := os.Open(file)
    // 处理文件
    f.Close()
}
```

### 4.2 defer 参数在声明时就确定了

```go
x := 1
defer fmt.Println(x) // 打印 1，不是 2
x = 2
```

**为什么：** defer 注册时会立刻计算参数的值（值拷贝），后面改 x 不影响。

**如果想要延迟取值：** 用闭包：

```go
x := 1
defer func() {
    fmt.Println(x) // 打印 2，闭包捕获的是变量引用
}()
x = 2
```

---

## 五、Interface 篇

### 5.1 接口的 nil 判断有坑

```go
type MyError struct{ msg string }
func (e *MyError) Error() string { return e.msg }

func doSomething() error {
    var err *MyError // nil 指针
    return err       // 返回了一个"装着 nil 指针的 interface"
}

func main() {
    err := doSomething()
    fmt.Println(err == nil) // false！
}
```

**为什么：** interface 内部有两个字段：类型信息 + 值。`return err` 返回的 interface 里类型是 `*MyError`、值是 nil。类型不为空，所以 interface 本身不等于 nil。

这是 Go 里最经典的坑之一。

**正确做法：** 要返回 nil error，直接 return nil：

```go
func doSomething() error {
    // 没出错
    return nil // 不要 return 一个类型化的 nil 指针
}
```

---

## 六、循环变量与指针篇

### 6.1 循环里取地址，全指向同一个变量

```go
type User struct{ Name string }
users := []User{{"张三"}, {"李四"}, {"王五"}}

var ptrs []*User
for _, u := range users {
    ptrs = append(ptrs, &u) // 所有指针都指向同一个 u
}
// ptrs 里三个指针，全部指向"王五"
```

**为什么：** `range` 的 `u` 是同一个变量，每次循环只是赋了新值。取地址拿到的始终是这个变量的地址。

**正确做法：** 用索引取原始元素的地址：

```go
for i := range users {
    ptrs = append(ptrs, &users[i]) // 指向原切片的元素
}
```

---

## 七、字符串与 range 篇

### 7.1 range 字符串拿到的是 rune 不是 byte

```go
s := "你好Go"
for i, ch := range s {
    fmt.Printf("i=%d ch=%c\n", i, ch)
}
// i=0 ch=你
// i=3 ch=好   ← 注意：i 跳了 3，不是 1
// i=6 ch=G
// i=7 ch=o
```

**为什么：** Go 字符串底层是 UTF-8 编码的字节数组。中文一个字占 3 个字节。`range` 按 rune（字符）遍历，index 是字节偏移量。

**注意点：**
- `len("你好")` 返回 6（6 个字节），不是 2（2 个字符）
- 想拿字符个数用 `utf8.RuneCountInString("你好")`
- 想按字节遍历用 `for i := 0; i < len(s); i++`

---

## 八、错误处理篇

### 8.1 error 被覆盖（shadow）

```go
func doWork() error {
    err := step1()
    if err != nil { return err }

    // 注意这里用了 :=
    result, err := step2() // 这个 err 是同一个变量（同作用域）OK

    if result > 0 {
        data, err := step3() // 这个 err 是新变量！外层的 err 没变
        if err != nil {
            return err // 这行能返回
        }
        process(data)
    }

    return err // step3 出错了，但这里的 err 还是 step2 的
}
```

**为什么：** `if` / `for` 大括号里用 `:=` 声明的变量是新的局部变量，跟外面的同名变量没关系。

**正确做法：** 在内层作用域用 `=` 而不是 `:=`，或者换个变量名：

```go
if result > 0 {
    data, err2 := step3() // 明确用不同名字
    if err2 != nil {
        return err2
    }
    process(data)
}
```

### 8.2 只判断 err != nil，忘了 err == nil 也可能有问题

```go
resp, err := http.Get(url)
if err != nil {
    return err
}
// 忘了 defer resp.Body.Close()
// resp.Body 不关的话连接不会归还连接池，连接泄漏
```

**正确做法：** HTTP 响应拿到后立刻 defer Close：

```go
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

## 九、Channel 篇

### 9.1 向已关闭的 channel 发数据直接 panic

```go
ch := make(chan int, 1)
close(ch)
ch <- 1 // panic: send on closed channel
```

**规则：**
- 关闭后可以读（读到零值 + false）
- 关闭后不能写（panic）
- 关闭后不能再关（panic）

**记住一句话：** 只有发送方才应该关闭 channel，接收方永远不要关。

### 9.2 读无缓冲 channel 没有发送方，永远卡死

```go
ch := make(chan int)
val := <-ch // 永远阻塞，没人发数据
```

这不会 panic，程序直接挂起不动。如果是在 main 里，Go 运行时会检测到死锁并报 `fatal error: all goroutines are asleep`。

---

## 十、JSON 篇

### 10.1 小写字段不会被序列化

```go
type User struct {
    name string // 小写开头
    Age  int
}
u := User{name: "张三", Age: 25}
data, _ := json.Marshal(u)
fmt.Println(string(data)) // {"Age":25}  name 没了
```

**为什么：** Go 里小写字段是包内私有的，`json.Marshal` 在 `encoding/json` 包里，访问不到你的私有字段。

**正确做法：** 要序列化的字段必须大写开头，用 tag 控制 JSON key：

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
```

### 10.2 time.Time 序列化格式问题

```go
type Event struct {
    CreatedAt time.Time `json:"created_at"`
}
// 序列化后："created_at":"2024-01-15T14:30:00.123456+08:00"
// 这是 RFC3339 格式，大部分前端能解析，但某些系统需要时间戳
```

**如果前端要时间戳：** 用自定义类型或者返回 Unix 时间戳：

```go
type Event struct {
    CreatedAt int64 `json:"created_at"` // time.Now().Unix()
}
```

---

## 总结速查表

| 类别 | 坑 | 一句话 |
|------|-----|--------|
| Slice | append 共享底层数组 | 不确定就先 copy |
| Slice | 子切片写穿 | 用三下标切片 `[a:b:c]` |
| Slice | nil vs 空 | 返给前端用 `make([]T, 0)` |
| Map | 并发读写 panic | 加锁或用 sync.Map |
| Map | nil map 写 panic | 用之前先 make |
| Goroutine | 闭包捕获循环变量 | 当参数传进去 |
| Goroutine | 泄漏 | channel 用带缓冲的，配合 context |
| Defer | 循环里 defer 堆积 | 包成匿名函数或手动关 |
| Defer | 参数声明时就确定 | 要延迟取值用闭包 |
| Interface | nil 指针装进 interface 不等于 nil | 直接 return nil |
| 循环 | range 取地址全指向最后一个 | 用索引 `&slice[i]` |
| 字符串 | range 按 rune 遍历 | len 是字节数不是字符数 |
| Error | 变量 shadow | 内层作用域别用 `:=` 覆盖外层 err |
| Channel | 关闭后写 panic | 只有发送方关 channel |
| JSON | 小写字段不序列化 | 大写 + json tag |

**核心原则：Go 的坑大部分来自三个地方——引用共享、并发竞争、隐式行为。写代码时多问自己：这块内存还有谁在用？这个变量是不是被别的 goroutine 碰了？这个操作的默认行为是不是我以为的那样？**
