package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	requiredCode = `
	if %v.%v == %v {
		return false
	}

`
	minIntCode = `
	if %v.%v < %v {
		return false
	}

`

	minStrCode = `
	if len(%v.%v) < %v {
		return false
	}

`

	maxIntCode = `
	if %v.%v > %v {
		return false
	}

`
	checkStringIsIntCode = `
	if _, err := strconv.Atoi(%v.%v); err != nil {
		return false
	}
`
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line

	imports := map[string]interface{}{}
	imports["net/http"] = true
	body := ""
	for _, topDecl := range node.Decls {
		switch declT := topDecl.(type) {
		case *ast.FuncDecl:
			continue
		case *ast.GenDecl:
			genNode := topDecl.(*ast.GenDecl)
			for _, spec := range genNode.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
					continue
				}

				structName := currType.Name.Name
				parseTemplate := fmt.Sprintf("func paramsParse%v(r *http.Request) (*%v, error) {\n", structName, structName)
				parseTemplate += fmt.Sprintf("	resStruct := &%v{}\n", structName)
				validateTemplate := fmt.Sprintf("func (s *%v) Validate() bool {\n", structName)
				isParamsStruct := false
			FIELDS_LOOP:
				for _, field := range currStruct.Fields.List {
					if field.Tag != nil {
						tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
						apiValidateStr := tag.Get("apivalidator")
						if apiValidateStr == "" {
							continue FIELDS_LOOP
						}
						isParamsStruct = true

						apiValidateOpts := strings.Split(apiValidateStr, ",")
						for _, opt := range apiValidateOpts {
							fieldType := field.Type.(*ast.Ident).Name
							if fieldType != "int" && fieldType != "string" {
								log.Fatalln("unsupported", fieldType)
							}

							if opt == "required" {
								if fieldType == "int" {
									validateTemplate += fmt.Sprintf(requiredCode, "s", field.Names[0].Name, 0)
								} else {
									validateTemplate += fmt.Sprintf(requiredCode, "s", field.Names[0].Name, `""`)
								}
								continue FIELDS_LOOP
							}

							parsedOpt := strings.Split(opt, "=")
							if len(parsedOpt) != 2 {
								log.Fatalln("need pair field", fieldType)
							}

							if parsedOpt[0] == "enum" {
								enumList := strings.Split(parsedOpt[1], "|")
								conditionList := []string{}
								if fieldType == "int" {
									for enumItem := range enumList {
										condition := fmt.Sprintf("s.%v != %v", field.Names[0].Name, enumItem)
										conditionList = append(conditionList, condition)
									}
								} else {
									for _, enumItem := range enumList {
										condition := fmt.Sprintf("s.%v != \"%v\"", field.Names[0].Name, enumItem)
										conditionList = append(conditionList, condition)
									}
								}
								validateTemplate += fmt.Sprintf("	if %v {\n", strings.Join(conditionList, " && "))
								validateTemplate += fmt.Sprintf("		return false\n	}\n\n")
							}

							if parsedOpt[0] == "min" {
								if _, err := strconv.Atoi(parsedOpt[1]); err != nil {
									log.Fatalln("min should be integer")
								}

								if fieldType == "int" {
									validateTemplate += fmt.Sprintf(minIntCode, "s", field.Names[0].Name, parsedOpt[1])
								} else {
									validateTemplate += fmt.Sprintf(minStrCode, "s", field.Names[0].Name, parsedOpt[1])
								}
							}

							if parsedOpt[0] == "max" {
								if _, err := strconv.Atoi(parsedOpt[1]); err != nil {
									log.Fatalln("max should be integer")
								}

								if fieldType == "int" {
									validateTemplate += fmt.Sprintf(maxIntCode, "s", field.Names[0].Name, parsedOpt[1])
								} else {
									log.Fatalln("max validator only for int params")
								}
							}
						}
					}
				}

				if isParamsStruct {
					parseTemplate += "	return resStruct, nil\n"
					parseTemplate += "}\n\n"
					body += parseTemplate
					validateTemplate += "	return true\n"
					validateTemplate += "}\n\n"
					body += validateTemplate
				}
			}
		default:
			fmt.Printf("SKIP %T is not *ast.FuncDecl or *ast.GenDecl\n", declT)
			continue
		}
	}
	if len(imports) > 0 {
		fmt.Fprint(out, "import (\n")
		for k := range imports {
			fmt.Fprintf(out, "	\"%v\"\n", k)
		}
		fmt.Fprint(out, ")\n\n")
	}
	fmt.Fprint(out, body)
}
