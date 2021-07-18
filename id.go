package jsonrpc

import (
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/goutlz/errz"
)

type IdFactory func() interface{}
type generateRawIdFunc func() (*rawId, error)

type rawId struct {
	StringValue      string
	ConvertedFromInt bool
}

func (b *rawId) UnmarshalJSON(data []byte) error {
	var rawInterface interface{}
	err := json.Unmarshal(data, &rawInterface)
	if err != nil {
		return err
	}

	if rawInterface == nil {
		return nil
	}

	switch t := rawInterface.(type) {
	case string:
		b.StringValue = rawInterface.(string)
		return nil
	case int64:
		b.StringValue = strconv.FormatInt(rawInterface.(int64), 10)
		b.ConvertedFromInt = true
		return nil
	case float64:
		b.StringValue = strconv.FormatInt(int64(rawInterface.(float64)), 10)
		b.ConvertedFromInt = true
		return nil
	default:
		return errz.Newf("Unsupported id type %T. Expected number or string", t)
	}
}

func (b rawId) MarshalJSON() ([]byte, error) {
	if !b.ConvertedFromInt {
		return json.Marshal(b.StringValue)
	}

	iVal, err := strconv.Atoi(b.StringValue)
	if err != nil {
		return nil, errz.Wrapf(err, "Failed to get int value from %s", b.StringValue)
	}

	return json.Marshal(iVal)
}

func (b *rawId) String() string {
	return b.StringValue
}

func defaultIdFactory() interface{} {
	return uuid.New().String()
}

func newGenerateIdFunc(factory IdFactory) generateRawIdFunc {
	return func() (*rawId, error) {
		idInterface := factory()

		switch t := idInterface.(type) {
		case string:
			return &rawId{
				StringValue: idInterface.(string),
			}, nil
		case int64:
			return &rawId{
				StringValue:      strconv.FormatInt(idInterface.(int64), 10),
				ConvertedFromInt: true,
			}, nil
		default:
			return nil, errz.Newf("Unsupported id type %T. Expected int64 or string", t)
		}
	}
}
