package gkit_gorm

import (
	"context"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
	"sync"
)

// BatchSave 提供了一个便捷的批量保存数据的方法，支持自动区分新增和更新操作
// 参数:
//   - db: GORM数据库连接
//   - data: 需要保存的数据集合，必须是切片或数组类型
//   - options: 可选的配置选项，用于自定义保存行为
//
// 返回:
//   - error: 操作过程中发生的错误，如果操作成功则返回nil
func BatchSave(db *gorm.DB, data any, options ...BatchSaveOption) error {
	// 初始化批量保存工具
	tool, err := newBatchSave(db, data, options...)
	if err != nil {
		return err
	}
	// 调用Save方法执行实际的保存操作
	return tool.Save()
}

// BatchSaveOption 定义了批量保存工具的函数式选项类型
// 支持的配置选项包括:
//   - BatchSize: 每批处理的数据量
//   - DuplicatedKey: 用于判断记录是否存在的键
//   - UpdateSelect: 更新时选择的字段
//   - CreateSelect: 创建时选择的字段
//   - Transaction: 是否使用事务
//   - 以及创建和更新时的字段忽略选项
type BatchSaveOption func(*batchSave)

// WithBatchSize 设置批量保存的批次大小，用于控制每次数据库操作的数据量
// 参数:
//   - size: 每批处理的记录数，必须大于0才会生效，否则使用默认值
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithBatchSize(size int) BatchSaveOption {
	return func(tool *batchSave) {
		if size > 0 {
			tool.BatchSize = size
		}
	}
}

// WithDuplicatedKey 设置用于判断数据库中记录是否已存在的字段
// 这些字段将用于构建查询条件，以确定记录应该被更新还是新建
// 参数:
//   - keys: 一个或多个字段名，用于唯一标识一条记录
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithDuplicatedKey(keys ...string) BatchSaveOption {
	return func(tool *batchSave) {
		if len(keys) > 0 {
			tool.DuplicatedKey = keys
		}
	}
}

// WithUpdateSelect 设置更新记录时需要更新的字段列表
// 只有指定的字段会在更新操作中被包含，其他字段将保持不变
// 参数:
//   - fields: 更新操作中需要包含的字段名列表
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithUpdateSelect(fields ...string) BatchSaveOption {
	return func(tool *batchSave) {
		if len(fields) > 0 {
			tool.UpdateSelect = fields
		}
	}
}

// WithCreateSelect 设置创建记录时需要包含的字段列表
// 只有指定的字段会在创建操作中被包含，其他字段将使用零值
// 参数:
//   - fields: 创建操作中需要包含的字段名列表
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithCreateSelect(fields ...string) BatchSaveOption {
	return func(tool *batchSave) {
		if len(fields) > 0 {
			tool.CreateSelect = fields
		}
	}
}

// WithTransaction 设置批量保存操作是否在事务中执行
// 在事务中执行可以确保数据一致性，但可能会影响性能
// 参数:
//   - transaction: 是否开启事务，true表示开启，false表示关闭
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithTransaction(transaction bool) BatchSaveOption {
	return func(tool *batchSave) {
		tool.Transaction = transaction
	}
}

// WithMaxRetryCount 设置处理重复键错误时的最大重试次数
// 防止因数据问题导致的无限循环
// 参数:
//   - count: 最大重试次数，必须大于0才会生效，否则使用默认值
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithMaxRetryCount(count int) BatchSaveOption {
	return func(tool *batchSave) {
		if count > 0 {
			tool.MaxRetryCount = count
		}
	}
}

// WithUpdateOmit 设置更新记录时需要忽略的字段列表
// 指定的字段在更新操作中将被排除，不会被修改
// 参数:
//   - fields: 更新操作中需要忽略的字段名列表
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithUpdateOmit(fields ...string) BatchSaveOption {
	return func(tool *batchSave) {
		if len(fields) > 0 && tool.ModelSchema != nil {
			// 从所有字段中排除需要忽略的字段
			allFields := getModelFields(tool.ModelSchema)
			tool.UpdateSelect = slice.Difference(allFields, fields)
		}
	}
}

// WithCreateOmit 设置创建记录时需要忽略的字段列表
// 指定的字段在创建操作中将被排除，将使用零值或数据库默认值
// 参数:
//   - fields: 创建操作中需要忽略的字段名列表
//
// 返回:
//   - BatchSaveOption: 返回一个可应用于BatchSaveTool的选项函数
func WithCreateOmit(fields ...string) BatchSaveOption {
	return func(tool *batchSave) {
		if len(fields) > 0 && tool.ModelSchema != nil {
			// 从所有字段中排除需要忽略的字段
			allFields := getModelFields(tool.ModelSchema)
			tool.CreateSelect = slice.Difference(allFields, fields)
		}
	}
}

// batchSave 批量保存工具结构体，用于执行批量保存操作
type batchSave struct {
	Database      *gorm.DB       // GORM数据库连接
	BatchSize     int            // 每个批次的大小，默认100
	ModelSchema   *schema.Schema // 模型的Schema信息
	Entities      []any          // 需要保存的实体集合
	DuplicatedKey []string       // 用于判断数据库中记录是否存在的键，用来决定执行更新还是创建操作
	UpdateSelect  []string       // 更新操作时包含的字段列表，默认是所有字段
	CreateSelect  []string       // 创建操作时包含的字段列表，默认是所有字段
	Transaction   bool           // 是否在事务中执行操作，默认为true
	MaxRetryCount int            // 处理重复键错误时的最大重试次数，默认为3次
}

// getModelFields 获取模型的所有数据库字段名
// 参数:
//   - modelSchema: 模型的Schema信息
//
// 返回:
//   - []string: 包含所有数据库字段名的切片
func getModelFields(modelSchema *schema.Schema) []string {
	fields := make([]string, 0, len(modelSchema.Fields))
	for _, field := range modelSchema.Fields {
		fields = append(fields, field.DBName)
	}
	return fields
}

// newBatchSave 创建并初始化一个批量保存工具实例
// 参数:
//   - db: GORM数据库连接
//   - data: 需要保存的数据集合，必须是切片或数组类型
//   - options: 可选的配置选项，用于自定义保存行为
//
// 返回:
//   - *batchSave: 初始化后的批量保存工具实例
//   - error: 初始化过程中发生的错误，如果成功则返回nil
func newBatchSave(db *gorm.DB, data any, options ...BatchSaveOption) (*batchSave, error) {
	// 1.初始化工具实例，设置默认值
	tool := &batchSave{
		Database:      db,
		BatchSize:     100,  // 默认批次大小为100
		Transaction:   true, // 默认开启事务
		MaxRetryCount: 3,    // 默认最大重试次数为3次
	}

	// 2.解析data，提取实体和模型类型
	// data是一个切片的模型，需要处理多种情况：指针类型的(切片/数组)模型，(切片/数组)的指针模型，(切片/数组)的模型
	entities, modelType, err := extractEntities(data)
	if err != nil {
		return nil, err
	}
	tool.Entities = entities

	// 3.使用GORM的schema包解析模型结构
	// 创建modelType的实例，因为schema.Parse需要的是实例而不是类型
	modelInstance := reflect.New(modelType).Interface()
	modelSchema, err := schema.Parse(modelInstance, &sync.Map{}, db.NamingStrategy)
	if err != nil {
		return nil, fmt.Errorf("解析模型失败: %w", err)
	}
	tool.ModelSchema = modelSchema

	// 4.根据schema解析的内容，设置默认配置
	// DuplicatedKey默认使用主键
	if modelSchema.PrioritizedPrimaryField != nil {
		tool.DuplicatedKey = []string{modelSchema.PrioritizedPrimaryField.DBName}
	}
	// 如果没有主键，DuplicatedKey保持为空，后面会校验

	// UpdateSelect和CreateSelect默认包含所有字段
	allFields := getModelFields(modelSchema)
	tool.UpdateSelect = allFields
	tool.CreateSelect = allFields

	// 5.应用函数选项配置，覆盖默认设置
	for _, option := range options {
		option(tool)
	}

	// 校验DuplicatedKey是否为空，这是必须的配置
	if len(tool.DuplicatedKey) == 0 {
		return nil, errors.New("DuplicatedKey不能为空")
	}

	return tool, nil
}

// Save 执行批量保存操作，自动处理创建和更新逻辑
// 根据配置决定是否在事务中执行，并将数据分批处理
// 返回:
//   - error: 保存过程中发生的错误，如果成功则返回nil
func (b *batchSave) Save() error {
	// 1.如果实体列表为空，无需执行任何操作，直接返回nil
	if len(b.Entities) == 0 {
		return nil
	}

	// 2.将实体列表按照批次大小进行分组
	batches := slice.Chunk(b.Entities, b.BatchSize)

	// 3.根据Transaction属性决定是否在事务中执行
	if b.Transaction {
		// 在事务中执行所有批次的处理
		return b.Database.Transaction(func(tx *gorm.DB) error {
			return b.processBatches(tx, batches)
		})
	}

	// 不使用事务直接处理批次
	return b.processBatches(b.Database, batches)
}

// processBatches 处理分批的数据，执行查询、更新和创建操作
// 参数:
//   - tx: GORM数据库连接或事务
//   - batches: 按批次分组的实体数据
//
// 返回:
//   - error: 处理过程中发生的错误，如果成功则返回nil
func (b *batchSave) processBatches(tx *gorm.DB, batches [][]any) error {
	// 遍历每个批次进行处理
	for _, batch := range batches {
		// 1.根据DuplicatedKey字段查询数据库中已存在的记录
		existMap, err := b.findExistingEntities(tx, batch)
		if err != nil {
			return err
		}

		// 2.根据查询结果，将实体分为需要更新和需要创建的两组
		updateEntities, createEntities := b.separateEntities(batch, existMap)

		// 3.处理需要更新的实体
		if len(updateEntities) > 0 {
			if err := b.updateEntities(tx, updateEntities); err != nil {
				return err
			}
		}

		// 4.处理需要创建的实体
		if len(createEntities) > 0 {
			// 循环处理重复键错误，直到没有错误或错误不是重复键错误
			// 这种情况可能发生在并发环境下，其他事务可能在我们查询后创建了相同的记录
			retryCount := 0
			for retryCount < b.MaxRetryCount {
				err := b.createEntities(tx, createEntities)
				if err == nil {
					break // 没有错误，跳出循环
				}

				// 检查是否是重复键错误
				isDuplicateKeyError := errors.Is(err, gorm.ErrDuplicatedKey)

				// 检查是否是MySQL的1062错误（重复键错误）
				var mysqlErr *mysql.MySQLError
				if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
					isDuplicateKeyError = true
				}

				// 如果不是任何形式的重复键错误，直接返回错误
				if !isDuplicateKeyError {
					return err
				}

				// 增加重试计数
				retryCount++

				// 处理重复键错误：可能是并发插入导致的
				// 重新查询存在的实体
				existMap, err := b.findExistingEntities(tx, createEntities)
				if err != nil {
					return err
				}

				// 重新分离需要更新和创建的实体
				updateEntities, newCreateEntities := b.separateEntities(createEntities, existMap)
				createEntities = newCreateEntities // 更新待创建实体列表

				// 更新那些本来要创建但现在已存在的实体
				if len(updateEntities) > 0 {
					if err := b.updateEntities(tx, updateEntities); err != nil {
						return err
					}
				}

				// 如果没有需要创建的实体了，跳出循环
				if len(createEntities) == 0 {
					break
				}
			}

			// 如果达到最大重试次数但仍有实体需要创建，返回最后一次的具体错误
			if retryCount >= b.MaxRetryCount && len(createEntities) > 0 {
				// 尝试最后一次创建，获取具体错误信息
				lastErr := b.createEntities(tx, createEntities)
				return fmt.Errorf("达到最大重试次数(%d)后仍有%d个实体未能成功创建: %w", b.MaxRetryCount, len(createEntities), lastErr)
			}
		}
	}

	return nil
}

// findExistingEntities 根据重复键查询数据库中已存在的实体
// 参数:
//   - tx: GORM数据库连接或事务
//   - entities: 需要检查的实体列表
//
// 返回:
//   - map[string]any: 以重复键生成的唯一标识为键，实体数据为值的映射
//   - error: 查询过程中发生的错误，如果成功则返回nil
func (b *batchSave) findExistingEntities(tx *gorm.DB, entities []any) (map[string]any, error) {
	// 如果实体列表为空或没有设置重复键，则返回空映射
	if len(entities) == 0 || len(b.DuplicatedKey) == 0 {
		return make(map[string]any), nil
	}

	// 1.从实体中提取重复键的值
	keyValues := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		keyValue := make(map[string]any)
		for _, key := range b.DuplicatedKey {
			val, err := getFieldValue(entity, b.ModelSchema, key)
			if err != nil {
				return nil, err
			}
			keyValue[key] = val
		}
		keyValues = append(keyValues, keyValue)
	}

	// 2.构建查询条件
	query := tx.Model(reflect.New(b.ModelSchema.ModelType).Interface())
	if len(b.DuplicatedKey) == 1 {
		// 单个键的情况，使用IN查询（更高效）
		key := b.DuplicatedKey[0]
		values := make([]any, 0, len(keyValues))
		for _, kv := range keyValues {
			values = append(values, kv[key])
		}
		query = query.Where(fmt.Sprintf("%s IN ?", key), values)
	} else {
		// 多个键的情况，使用OR和AND组合查询
		// 例如：(key1 = ? AND key2 = ?) OR (key1 = ? AND key2 = ?)
		var conditions []string
		var values []any
		for _, kv := range keyValues {
			condition := make([]string, 0, len(b.DuplicatedKey))
			for _, key := range b.DuplicatedKey {
				condition = append(condition, fmt.Sprintf("%s = ?", key))
				values = append(values, kv[key])
			}
			conditions = append(conditions, "("+strings.Join(condition, " AND ")+")")
		}
		query = query.Where(strings.Join(conditions, " OR "), values...)
	}

	// 3.执行查询获取已存在的实体
	var existingEntities []map[string]any
	if err := query.Find(&existingEntities).Error; err != nil {
		return nil, err
	}

	// 4.构建以重复键为索引的映射，方便快速查找
	existMap := make(map[string]any)
	for _, entity := range existingEntities {
		key := generateKey(entity, b.DuplicatedKey)
		existMap[key] = entity
	}

	return existMap, nil
}

// separateEntities 将实体分为需要更新和需要创建的两组
// 参数:
//   - entities: 需要处理的实体列表
//   - existMap: 数据库中已存在实体的映射
//
// 返回:
//   - []any: 需要更新的实体列表
//   - []any: 需要创建的实体列表
func (b *batchSave) separateEntities(entities []any, existMap map[string]any) ([]any, []any) {
	// 初始化更新和创建实体的切片
	updateEntities := make([]any, 0)
	createEntities := make([]any, 0)

	// 遍历每个实体，根据是否在existMap中存在决定是更新还是创建
	for _, entity := range entities {
		// 提取实体的重复键值
		keyValues := make(map[string]any)
		for _, key := range b.DuplicatedKey {
			val, _ := getFieldValue(entity, b.ModelSchema, key)
			keyValues[key] = val
		}

		// 生成唯一键并检查是否存在
		key := generateKey(keyValues, b.DuplicatedKey)
		if _, exists := existMap[key]; exists {
			// 如果存在，则添加到更新列表
			updateEntities = append(updateEntities, entity)
		} else {
			// 如果不存在，则添加到创建列表
			createEntities = append(createEntities, entity)
		}
	}

	return updateEntities, createEntities
}

// updateEntities 更新数据库中已存在的实体
// 参数:
//   - tx: GORM数据库连接或事务
//   - entities: 需要更新的实体列表
//
// 返回:
//   - error: 更新过程中发生的错误，如果成功则返回nil
func (b *batchSave) updateEntities(tx *gorm.DB, entities []any) error {
	// 遍历每个需要更新的实体
	for _, entity := range entities {
		// 1.构建更新条件，基于重复键字段
		conditions := make([]string, 0, len(b.DuplicatedKey))
		values := make([]any, 0, len(b.DuplicatedKey))
		for _, key := range b.DuplicatedKey {
			val, err := getFieldValue(entity, b.ModelSchema, key)
			if err != nil {
				return err
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			values = append(values, val)
		}

		// 2.执行更新操作
		// 使用Select指定要更新的字段，避免更新所有字段
		query := tx.Model(entity).Select(b.UpdateSelect)
		// 使用Where指定更新条件
		query = query.Where(strings.Join(conditions, " AND "), values...)
		// 执行更新并检查错误
		if err := query.Updates(entity).Error; err != nil {
			return err
		}
	}

	return nil
}

// createEntities 在数据库中创建新实体
// 参数:
//   - tx: GORM数据库连接或事务
//   - entities: 需要创建的实体列表
//
// 返回:
//   - error: 创建过程中发生的错误，如果成功则返回nil
func (b *batchSave) createEntities(tx *gorm.DB, entities []any) error {
	// 如果实体列表为空，无需执行任何操作
	if len(entities) == 0 {
		return nil
	}

	// 1.创建模型实例，用于设置表名和其他模型级别的配置
	modelInstance := reflect.New(b.ModelSchema.ModelType).Interface()

	// 2.创建与模型类型匹配的切片，用于批量创建
	// 使用反射创建正确类型的切片，确保GORM可以正确处理
	sliceType := reflect.SliceOf(reflect.PointerTo(b.ModelSchema.ModelType))
	sliceValue := reflect.MakeSlice(sliceType, 0, len(entities))

	// 3.将entities中的元素转换为正确的类型并添加到新切片中
	for _, entity := range entities {
		// 获取entity的反射值
		entityValue := reflect.ValueOf(entity)
		// 添加到新切片
		sliceValue = reflect.Append(sliceValue, entityValue)
	}

	// 4.将新切片转换为interface{}
	typedEntities := sliceValue.Interface()

	// 5.执行批量创建操作
	// 使用Select指定要创建的字段，使用CreateInBatches进行批量创建
	return tx.Model(modelInstance).Select(b.CreateSelect).CreateInBatches(typedEntities, b.BatchSize).Error
}

// getFieldValue 从实体中获取指定字段的值
// 参数:
//   - entity: 实体对象
//   - modelSchema: 模型的Schema信息
//   - fieldName: 需要获取值的字段名（数据库字段名）
//
// 返回:
//   - any: 字段的值
//   - error: 获取过程中发生的错误，如果成功则返回nil
func getFieldValue(entity any, modelSchema *schema.Schema, fieldName string) (any, error) {
	// 获取实体的反射值，如果是指针则获取其元素
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 从模型Schema中查找字段
	field, ok := modelSchema.FieldsByDBName[fieldName]
	if !ok {
		return nil, fmt.Errorf("字段 %s 不存在", fieldName)
	}

	// 使用GORM的Field.ValueOf方法获取字段值
	fieldVal, _ := field.ValueOf(context.Background(), val)
	return fieldVal, nil
}

// generateKey 根据指定的键生成实体的唯一标识字符串
// 参数:
//   - entity: 包含字段值的映射
//   - keys: 用于生成唯一标识的键列表
//
// 返回:
//   - string: 由键值组合生成的唯一标识字符串
func generateKey(entity map[string]any, keys []string) string {
	// 为每个键提取值并转换为字符串
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		// 使用%v格式化任意类型的值
		parts = append(parts, fmt.Sprintf("%v", entity[key]))
	}
	// 使用下划线连接所有部分，形成唯一键
	return strings.Join(parts, "_")
}

// extractEntities 从输入数据中提取实体切片和模型类型
// 参数:
//   - data: 输入数据，必须是切片或数组类型
//
// 返回:
//   - []any: 提取的实体列表
//   - reflect.Type: 实体的模型类型
//   - error: 提取过程中发生的错误，如果成功则返回nil
func extractEntities(data any) ([]any, reflect.Type, error) {
	// 获取数据的反射值，如果是指针则获取其元素
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 检查数据是否是切片或数组类型
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, nil, errors.New("数据必须是切片或数组")
	}

	// 获取元素类型
	var elemType reflect.Type
	if val.Len() > 0 {
		// 如果切片不为空，从第一个元素获取类型
		elemVal := val.Index(0)
		if elemVal.Kind() == reflect.Ptr {
			elemType = elemVal.Elem().Type()
		} else {
			elemType = elemVal.Type()
		}
	} else {
		// 如果切片为空，尝试从切片类型获取元素类型
		sliceType := val.Type()
		elemType = sliceType.Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
	}

	// 提取所有实体到一个统一的切片中
	entities := make([]any, val.Len())
	for i := 0; i < val.Len(); i++ {
		entities[i] = val.Index(i).Interface()
	}

	return entities, elemType, nil
}
