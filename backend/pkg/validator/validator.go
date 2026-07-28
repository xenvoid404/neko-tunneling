package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{validate: v}
}

func (vl *Validator) ValidateStruct(s interface{}) map[string][]string {
	err := vl.validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string][]string{"_": {"validasi gagal: " + err.Error()}}
	}

	errors := make(map[string][]string)
	for _, fe := range validationErrs {
		field := displayName(fe.Field())
		errors[field] = append(errors[field], parseErrorMessage(fe))
	}

	return errors
}

func displayName(field string) string {
	if field == "" {
		return field
	}
	return strings.ToLower(field[:1]) + field[1:]
}

func parseErrorMessage(err validator.FieldError) string {
	field := displayName(err.Field())
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", field)
	case "min":
		return fmt.Sprintf("%s minimal %s", field, err.Param())
	case "max":
		return fmt.Sprintf("%s maksimal %s", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari: %s", field, err.Param())
	case "alphanum":
		return fmt.Sprintf("%s hanya boleh berisi huruf dan angka", field)
	case "gte":
		return fmt.Sprintf("%s harus lebih besar dari %s", field, err.Param())
	default:
		return fmt.Sprintf("%s tidak valid", field)
	}
}
