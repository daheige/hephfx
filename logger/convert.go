package logger

import (
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// FieldToValue 将zap field类型转换为 interface
func FieldToValue(f zap.Field) interface{} {
	switch f.Type {
	case zapcore.BinaryType:
		return f.Interface.([]byte)
	case zapcore.BoolType:
		return f.Integer == 1
	case zapcore.ByteStringType:
		return f.Interface.([]byte)
	case zapcore.Complex128Type:
		return f.Interface.(complex128)
	case zapcore.Complex64Type:
		return f.Interface.(complex64)
	case zapcore.DurationType:
		return time.Duration(f.Integer)
	case zapcore.Float64Type:
		return math.Float64frombits(uint64(f.Integer))
	case zapcore.Float32Type:
		return math.Float32frombits(uint32(f.Integer))
	case zapcore.Int64Type:
		return f.Integer
	case zapcore.Int32Type:
		return int32(f.Integer)
	case zapcore.Int16Type:
		return int16(f.Integer)
	case zapcore.Int8Type:
		return int8(f.Integer)
	case zapcore.StringType:
		return f.String
	case zapcore.TimeType:
		if f.Interface != nil {
			return time.Unix(0, f.Integer).In(f.Interface.(*time.Location))
		}

		// Fall back to UTC if location is nil
		return time.Unix(0, f.Integer)
	case zapcore.TimeFullType:
		return f.Interface.(time.Time)
	case zapcore.Uint64Type:
		return uint64(f.Integer)
	case zapcore.Uint32Type:
		return uint32(f.Integer)
	case zapcore.Uint16Type:
		return uint16(f.Integer)
	case zapcore.Uint8Type:
		return uint8(f.Integer)
	case zapcore.UintptrType:
		return uintptr(f.Integer)
	case zapcore.ReflectType, zapcore.StringerType:
		return f.Interface
	case zapcore.ErrorType:
		err, ok := f.Interface.(error)
		if ok && err != nil {
			return err.Error()
		}

		return fmt.Sprintf("%v", f.Interface)
	case zapcore.SkipType:
		break
	default:
	}

	return fmt.Sprintf("unknown field type: %v", f)
}

// MaskString 对字符串进行打码：保留前3位和后4位，中间替换为*
// 适用于长度在 7 位及以上的字符串（涵盖8-12位场景）
func MaskString(s string) string {
	if strings.Contains(s, "@") {
		return MaskAllString(s)
	}

	// 1.将字符串转换为 rune 切片，确保正确处理多字节字符（如中文）
	runes := []rune(s)
	length := len(runes)

	// 2.边界检查：如果长度不足7位（3+4），无法同时保留前后，根据业务需求处理
	// 此处策略：若长度小于等于7，全部打码或返回原串（视具体安全要求而定，这里选择全部打码以防泄露）
	if length <= 7 {
		return strings.Repeat("*", length)
	}

	// 3.计算中间需要替换的字符数量
	// 总长度 - 前3位 - 后4位
	maskCount := length - 7

	// 4.对于12位的字符串，不足12位的全部用*补全
	if maskCount < 5 {
		maskCount = 5
	}

	// 5.如果不超过32位的，前3后4，其他全部使用*替代
	if maskCount <= 25 {
		return string(runes[:3]) + strings.Repeat("*", maskCount) + string(runes[length-4:])
	}

	// 6.如果超过32位的，仅保留后4位，其他全部使用*替代
	return strings.Repeat("*", maskCount+3) + string(runes[length-4:])
}

// MaskAllString 全部打码
func MaskAllString(s string) string {
	// 1.将字符串转换为 rune 切片，确保正确处理多字节字符（如中文）
	runes := []rune(s)
	length := len(runes)

	// 2.边界检查：如果长度不足7位（3+4），无法同时保留前后，根据业务需求处理
	// 此处策略：若长度小于等于7，全部打码或返回原串（视具体安全要求而定，这里选择全部打码以防泄露）
	if length <= 7 {
		return strings.Repeat("*", length)
	}

	// 3.超过7位以上全部打码
	return strings.Repeat("*", length)
}
