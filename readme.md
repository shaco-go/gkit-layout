### 导入包
```shell
go get -u github.com/cockroachdb/errors
go get -u github.com/pkg/errors
go get -u github.com/duke-git/lancet/v2 
go get -u gorm.io/driver/mysql
go get -u gopkg.in/natefinch/lumberjack.v2
go get -u gorm.io/gorm
go get -u github.com/rs/zerolog
go get -u github.com/spf13/viper
```

## 项目说明

这是一个基于 Go 语言的轻量级开发框架，提供了常用的组件和工具，帮助开发者快速构建高效、可靠的应用程序。

### 主要功能

1. **配置管理**：使用 Viper 管理配置文件，支持 YAML、JSON 等格式
2. **日志系统**：基于 zerolog 的高性能日志系统，支持多种输出方式和格式
3. **数据库支持**：集成 GORM，支持 MySQL 等数据库
4. **缓存系统**：支持内存缓存和 Redis 缓存

### 日志配置说明

日志系统基于 zerolog 实现，提供了灵活的配置方式：

#### 基本配置

在 `configs/development.yaml` 或 `configs/production.yaml` 中配置：

```yaml
env: development  # 环境：development 或 production
app_name: app     # 应用名称
log_level: trace  # 日志级别：trace, debug, info, warn, error, fatal, panic
```

#### 高级配置

可以通过代码方式进行更详细的配置：

```go
import (
    gkit_zerolog "github.com/shaco-go/gkit-layout/pkg/zerolog"
)

// 创建默认配置
config := gkit_zerolog.DefaultLogConfig()

// 或者创建开发环境配置
config := gkit_zerolog.NewDevLogConfig()

// 或者创建生产环境配置
config := gkit_zerolog.NewProdLogConfig()

// 自定义配置
config.Level = zerolog.InfoLevel           // 设置日志级别
config.Output = gkit_zerolog.MultiOutput   // 设置输出类型：ConsoleOutput, FileOutput, MultiOutput
config.HumanReadable = true                // 是否使用人类可读格式
config.LogDir = "custom_logs"              // 自定义日志目录
config.LogFileName = "custom.log"          // 自定义日志文件名
config.MaxSize = 200                       // 单个日志文件最大大小，单位MB
config.MaxBackups = 10                     // 保留的旧文件最大数量
config.MaxAge = 30                         // 保留的最大天数
config.Compress = true                     // 是否压缩

// 创建日志实例
logger := gkit_zerolog.New(config)

// 添加常用字段
logger = logger.With().Stack().Caller().Timestamp().Logger()
```

#### 输出类型

日志系统支持三种输出类型：

1. **ConsoleOutput**：输出到控制台，适合开发环境
2. **FileOutput**：输出到文件，适合生产环境
3. **MultiOutput**：同时输出到控制台和文件，适合需要双重记录的场景

#### 人类可读格式

当 `HumanReadable` 设置为 `true` 时，日志将以更易于阅读的格式输出，包括彩色输出和更好的结构化展示。适合开发环境使用。

当设置为 `false` 时，日志将以 JSON 格式输出，适合生产环境和日志分析系统处理。