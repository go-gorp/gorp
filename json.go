package gorp

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func newJsonScanner(target interface{}) CustomScanner {
	return CustomScanner{
		Holder: new([]byte),
		Target: target,
		Binder: func(holder, target interface{}) error {
			sptr := holder.(*[]byte)
			if *sptr == nil {
				target_value := reflect.ValueOf(target).Elem()
				target_type := target_value.Type()
				if target_type.Kind() != reflect.Ptr {
					return fmt.Errorf("gorp: select of json null value requires pointer struct field, got %s", target_type.String())
				}
				target_value.Set(reflect.Zero(target_type))
				return nil
			}
			err := json.Unmarshal(*sptr, target)
			return err
		},
	}
}
