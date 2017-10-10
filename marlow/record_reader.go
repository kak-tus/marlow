package marlow

import "io"
import "fmt"
import "bytes"
import "go/ast"
import "regexp"
import "reflect"
import "net/url"
import "strings"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/features"
import "github.com/dadleyy/marlow/marlow/constants"

const (
	// DefaultBlueprintLimit is the default limit that will be used in blueprints unless one is configured on the record.
	DefaultBlueprintLimit = 100
)

func newRecordConfig(typeName string) url.Values {
	config := make(url.Values)
	config.Set(constants.RecordNameConfigOption, typeName)
	tableName := strings.ToLower(inflector.Pluralize(typeName))
	config.Set(constants.TableNameConfigOption, tableName)
	config.Set(constants.DefaultLimitConfigOption, fmt.Sprintf("%d", DefaultBlueprintLimit))
	storeName := fmt.Sprintf("%sStore", typeName)
	config.Set(constants.StoreNameConfigOption, storeName)
	config.Set(constants.BlueprintRangeFieldSuffixConfigOption, "Range")
	config.Set(constants.BlueprintLikeFieldSuffixConfigOption, "Like")
	config.Set(constants.StoreFindMethodPrefixConfigOption, "Find")
	config.Set(constants.StoreCountMethodPrefixConfigOption, "Count")
	return config
}

func parseStruct(d ast.Decl) (*ast.StructType, string, bool) {
	decl, ok := d.(*ast.GenDecl)

	if !ok {
		return nil, "", false
	}

	typeDecl, ok := decl.Specs[0].(*ast.TypeSpec)

	if !ok {
		return nil, "", false
	}

	structType, ok := typeDecl.Type.(*ast.StructType)

	if !ok {
		return nil, "", false
	}

	typeName := typeDecl.Name.String()
	return structType, typeName, true
}

func newRecordReader(root ast.Decl, imports chan<- string) (io.Reader, bool) {
	structType, typeName, ok := parseStruct(root)

	if !ok {
		return nil, false
	}

	recordConfig, recordFields := newRecordConfig(typeName), make(map[string]url.Values)

	for _, f := range structType.Fields.List {
		if f.Tag == nil {
			continue
		}

		tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
		fieldConfig, e := url.ParseQuery(tag.Get("marlow"))

		if e != nil || len(f.Names) == 0 {
			continue
		}

		name := f.Names[0].String()

		if name == "table" || name == "_" {
			for k := range fieldConfig {
				v := fieldConfig.Get(k)
				recordConfig.Set(k, v)
			}

			continue
		}

		if fieldConfig.Get("column") == "" {
			fieldConfig.Set("column", strings.ToLower(name))
		}

		// Convert our field's type to it's string counterpart.
		fieldType := fmt.Sprintf("%v", f.Type)

		// Check to see if this field is a complex type - one that refers to an exported type from another package.
		selector, ok := f.Type.(*ast.SelectorExpr)

		// If the field is a complex type, make an note of the import that it is referring to - this will be mapped to the
		// original import path from the source package by our import processor.
		if ok {
			fieldType = fmt.Sprintf("%s.%s", selector.X, selector.Sel)
			fieldConfig.Set("import", fmt.Sprintf("%s", selector.X))
		}

		fieldConfig.Set("type", fieldType)
		recordFields[name] = fieldConfig
	}

	pr, pw := io.Pipe()

	if v := regexp.MustCompile("^[A-z_]+$"); v.MatchString(recordConfig.Get("tableName")) != true {
		pw.CloseWithError(fmt.Errorf("invalid-table"))
		return pr, true
	}

	go func() {
		e := readRecord(pw, recordConfig, recordFields, imports)
		pw.CloseWithError(e)
	}()

	return pr, true
}

func readRecord(writer io.Writer, config url.Values, fields map[string]url.Values, imports chan<- string) error {
	buffer := new(bytes.Buffer)
	readers, enabled := make([]io.Reader, 0), make(map[string]bool)

	for _, fieldConfig := range fields {
		queryable := fieldConfig.Get(constants.QueryableConfigOption)
		updateable := fieldConfig.Get(constants.UpdateableConfigOption)

		if _, e := enabled[constants.QueryableConfigOption]; queryable != "false" && !e {
			generator := features.NewQueryableGenerator(config, fields, imports)
			readers = append(readers, generator)
			enabled[constants.QueryableConfigOption] = true
		}

		if _, e := enabled[constants.UpdateableConfigOption]; updateable != "false" && !e {
			generator := features.NewUpdateableGenerator(config, fields, imports)
			readers = append(readers, generator)
			enabled[constants.UpdateableConfigOption] = true
		}
	}

	if len(readers) == 0 {
		comment := strings.NewReader(
			fmt.Sprintf("// [marlow no-features]: %s\n", config.Get(constants.RecordNameConfigOption)),
		)

		_, e := io.Copy(writer, comment)
		return e
	}

	// If we had any features enabled, we need to also generate the blue print API.
	readers = append(
		readers,
		features.NewStoreGenerator(config, imports),
		features.NewBlueprintGenerator(config, fields, imports),
	)

	// Iterate over all our collected features, copying them into the buffer
	if _, e := io.Copy(buffer, io.MultiReader(readers...)); e != nil {
		return e
	}

	_, e := io.Copy(writer, buffer)
	return e
}
