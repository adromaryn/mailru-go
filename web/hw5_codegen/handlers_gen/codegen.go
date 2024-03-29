package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	parseFuncCode = `
func paramsParse%v(r *http.Request) (res *%v, err error) {
	res = &%v{}
%v

	return res, err
}

`

	parseIntParamCode = `
res.%v, err = strconv.Atoi(r.FormValue("%v"))
if err != nil {
	return nil, errors.New("%v must be int")
}`

	parseStringParamCode = `
res.%v = r.FormValue("%v")`

	requiredCode = `
if s.%v == %v {
	return errors.New("%v must be not empty")
}`

	validateFuncCode = `
func (s *%v) Validate() (err error) {
%v

	return err
}`

	enumCheckCode = `
if %v {
	return errors.New("%v must be one of [%v]")
}`

	minIntCode = `
if s.%v < %v {
	return errors.New("%v must be >= %v")
}`

	minStrCode = `
if len(s.%v) < %v {
	return errors.New("%v len must be >= %v")
}`

	maxIntCode = `
if s.%v > %v {
	return errors.New("%v must be <= %v")
}`
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

type ApiDefinition struct {
	Url    string `json:"url"`
	Method string `json:"method"`
	Auth   bool   `json:"auth"`
}

type MethodDefinition struct {
	Auth      bool
	ParamName string
	FuncName  string
}

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
	var parseFuncs []string
	var validateFuncs []string
	// [type]->[url]->[method]->has_auth
	apiMethodDecls := make(map[string]map[string]map[string]MethodDefinition)
SPECS_LOOP:
	for _, topDecl := range node.Decls {
		switch declT := topDecl.(type) {
		case *ast.FuncDecl:
			funcNode := topDecl.(*ast.FuncDecl)
			if funcNode.Doc == nil {
				fmt.Printf("SKIP function %#v does not have comments\n", funcNode.Name.Name)
				continue SPECS_LOOP
			}

			needCodegen := false
			var apiDefinition *ApiDefinition
			for _, comment := range funcNode.Doc.List {
				if strings.HasPrefix(comment.Text, "// apigen:api") {
					needCodegen = true
					definitionStr := strings.TrimSpace(strings.TrimPrefix(comment.Text, "// apigen:api"))
					json.Unmarshal([]byte(definitionStr), &apiDefinition)
					break
				}
			}
			if !needCodegen {
				fmt.Printf("SKIP function %#v doesnt have apigen mark\n", funcNode.Name.Name)
				continue SPECS_LOOP
			}

			funcRecv := funcNode.Recv
			if funcRecv == nil {
				log.Fatalln("only methods for apigen")
			}
			star, ok := funcRecv.List[0].Type.(*ast.StarExpr)
			if !ok {
				log.Fatalln("api method receiver isnt pointer")
			}
			receiverTypeName := star.X.(*ast.Ident).Name

			if len(funcNode.Type.Params.List) != 2 {
				log.Fatalln("should has ctx and 'Params' args")
			}
			paramName := funcNode.Type.Params.List[1].Type.(*ast.Ident).Name

			if apiMethodDecls[receiverTypeName] == nil {
				apiMethodDecls[receiverTypeName] = make(map[string]map[string]MethodDefinition)
			}
			if apiMethodDecls[receiverTypeName][apiDefinition.Url] == nil {
				apiMethodDecls[receiverTypeName][apiDefinition.Url] = make(map[string]MethodDefinition)
			}
			apiMethodDecls[receiverTypeName][apiDefinition.Url][apiDefinition.Method] = MethodDefinition{apiDefinition.Auth, paramName, funcNode.Name.Name}
			imports["fmt"] = true
			imports["encoding/json"] = true

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
				if parseFunc := getParseFunc(structName, currStruct, imports); parseFunc != "" {
					parseFuncs = append(parseFuncs, parseFunc)
				}
				if validateFunc := getValidateFunc(structName, currStruct, imports); validateFunc != "" {
					validateFuncs = append(validateFuncs, validateFunc)
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

	fmt.Fprintln(out, strings.Join(parseFuncs, "\n"))
	fmt.Fprintln(out, strings.Join(validateFuncs, "\n"))
	fmt.Fprintln(out, parseApiDecls(&apiMethodDecls))
}

func parseApiDecls(apiMethodDecls *map[string]map[string]map[string]MethodDefinition) string {
	result := ""
	for apiName := range *apiMethodDecls {
		result += fmt.Sprintf("func (h *%v) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", apiName)
		result += `
	switch r.URL.Path {`

		urlDecls := (*apiMethodDecls)[apiName]
		for url := range urlDecls {
			result += fmt.Sprintf(`
	case %#v:`, url)
			methodDecls := urlDecls[url]
			result += parseMethodsDecls(&methodDecls)
		}
		result += `
	default:
		http.Error(w, "{\"error\": \"unknown method\"}", http.StatusNotFound)
	}

}

`
	}
	return result
}

func parseMethodsDecls(methodDecls *map[string]MethodDefinition) string {
	result := ""
	if len(*methodDecls) == 1 {
		var method string
		for method = range *methodDecls {
		}
		if method != "" {
			result += fmt.Sprintf(`
		if r.Method != %#v {
			http.Error(w, "{\"error\": \"bad method\"}", http.StatusNotAcceptable)
			return
		}
`, method)
		}
		result += parseMethod((*methodDecls)[method])
	} else {
		result += fmt.Sprintf(`
		switch r.Method {
`)
		for method, methodDef := range *methodDecls {
			if method == "" {
				continue
			}

			result += fmt.Sprintf(`
		case %v:
`, method)
			result += parseMethod(methodDef)
		}

		if methodDef, ok := (*methodDecls)[""]; ok {
			result += fmt.Sprintf(`
		default:
`)
			result += parseMethod(methodDef)
		}
	}
	return result
}

func parseMethod(method MethodDefinition) string {
	result := ""
	if method.Auth {
		result += `
		if r.Header.Get("X-Auth") != "100500" {
			http.Error(w, "{\"error\": \"unauthorized\"}", http.StatusForbidden)
			return
		}
`
	}

	result += fmt.Sprintf("\n		params, err := paramsParse%v(r)", method.ParamName)
	result += fmt.Sprint("\n		var errResp []byte")
	result += fmt.Sprint("\n		if err != nil {\n			errResp, _ = json.Marshal(map[string]interface{}{\"error\": err.Error()})\n			http.Error(w, string(errResp), http.StatusBadRequest)\n			return\n		}\n")

	result += "\n		err = params.Validate()"
	result += fmt.Sprintf("\n		if err != nil {\n			errResp, _ = json.Marshal(map[string]interface{}{\"error\": err.Error()})\n			http.Error(w, string(errResp), http.StatusBadRequest)\n			return\n		}\n")

	result += fmt.Sprintf("\n		fRes, err := h.%v(r.Context(), *params)\n", method.FuncName)
	result += fmt.Sprint("\n		if err != nil {")
	result += fmt.Sprint("\n			errApi, ok := err.(ApiError)")
	result += fmt.Sprint("\n			if(ok) {\n				errResp, _ = json.Marshal(map[string]interface{}{\"error\": errApi.Err.Error()})\n				http.Error(w, string(errResp), errApi.HTTPStatus)\n				return\n			} else {")
	result += fmt.Sprint("\n				errResp, _ = json.Marshal(map[string]interface{}{\"error\": err.Error()})\n				http.Error(w, string(errResp), http.StatusInternalServerError)\n				return\n			}\n")
	result += fmt.Sprint("\n		}")
	result += fmt.Sprint("\n		result := map[string]interface{}{\"response\": fRes, \"error\": \"\"}")
	result += fmt.Sprint("\n		resultMarhalled, err := json.Marshal(result)")
	result += fmt.Sprint("\n		if err != nil {\n			http.Error(w, \"{\\\"error\\\":\\\"\\\"}\", http.StatusInternalServerError)\n			return\n		}\n")

	result += fmt.Sprint("\n		fmt.Fprintln(w, string(resultMarhalled))")

	return result
}

func getParseFunc(structName string, paramsType *ast.StructType, imports map[string]interface{}) string {
	var paramsSetTemplates []string

	isParamsStruct := false
FIELDS_LOOP:
	for _, field := range paramsType.Fields.List {
		if field.Tag != nil {
			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			apiValidateStr := tag.Get("apivalidator")
			if apiValidateStr == "" {
				continue FIELDS_LOOP
			}
			isParamsStruct = true

			apiValidateOpts := strings.Split(apiValidateStr, ",")

			fieldType := field.Type.(*ast.Ident).Name
			if fieldType != "int" && fieldType != "string" {
				log.Fatalln("unsupported", fieldType)
			}

			var paramName string
			for _, opt := range apiValidateOpts {

				parsedOpt := strings.Split(opt, "=")
				if len(parsedOpt) == 1 {
					continue
				}
				if len(parsedOpt) != 2 {
					log.Fatalln("need pair field", fieldType)
				}

				if parsedOpt[0] == "paramname" {
					paramName = parsedOpt[1]
				}
			}
			if paramName == "" {
				paramName = toSnakeCase(field.Names[0].Name)
			}

			if fieldType == "int" {
				imports["strconv"] = true
				imports["errors"] = true
				paramsSetTemplates = append(paramsSetTemplates, fmt.Sprintf(parseIntParamCode, field.Names[0].Name, paramName, paramName))
			} else {
				paramsSetTemplates = append(paramsSetTemplates, fmt.Sprintf(parseStringParamCode, field.Names[0].Name, paramName))
			}
		}
	}

	if !isParamsStruct {
		return ""
	}

	funcBody := strings.Join(paramsSetTemplates, "")
	return fmt.Sprintf(parseFuncCode, structName, structName, structName, addSpace(funcBody, 1))
}

func getValidateFunc(structName string, paramsType *ast.StructType, imports map[string]interface{}) string {
	var validateParamsTemplates []string

	isParamsStruct := false
FIELDS_LOOP:
	for _, field := range paramsType.Fields.List {
		if field.Tag != nil {
			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			apiValidateStr := tag.Get("apivalidator")
			if apiValidateStr == "" {
				continue FIELDS_LOOP
			}
			isParamsStruct = true

			apiValidateOpts := strings.Split(apiValidateStr, ",")

			fieldType := field.Type.(*ast.Ident).Name
			if fieldType != "int" && fieldType != "string" {
				log.Fatalln("unsupported", fieldType)
			}

			var paramName string
			for _, opt := range apiValidateOpts {

				parsedOpt := strings.Split(opt, "=")
				if len(parsedOpt) == 1 {
					continue
				}
				if len(parsedOpt) != 2 {
					log.Fatalln("need pair field", fieldType)
				}

				if parsedOpt[0] == "paramname" {
					paramName = parsedOpt[1]
				}
			}
			if paramName == "" {
				paramName = toSnakeCase(field.Names[0].Name)
			}

		OPTS_LOOP:
			for _, opt := range apiValidateOpts {

				parsedOpt := strings.Split(opt, "=")
				if parsedOpt[0] == "required" {
					imports["errors"] = true
					if fieldType == "int" {
						validateParamsTemplates = append(validateParamsTemplates, fmt.Sprintf(requiredCode, field.Names[0].Name, 0, paramName))
					} else {
						validateParamsTemplates = append(validateParamsTemplates, fmt.Sprintf(requiredCode, field.Names[0].Name, `""`, paramName))
					}
					continue OPTS_LOOP
				}

				if len(parsedOpt) != 2 {
					log.Fatalln("need pair field", field.Names[0].Name)
				}

				if parsedOpt[0] == "enum" {
					imports["errors"] = true
					enumList := strings.Split(parsedOpt[1], "|")
					validateParamsTemplates = append(validateParamsTemplates, getEnumCheck(enumList, field.Names[0].Name, paramName, fieldType))
					continue OPTS_LOOP
				}

				if parsedOpt[0] == "min" {
					if _, err := strconv.Atoi(parsedOpt[1]); err != nil {
						log.Fatalln("min should be integer")
					}

					imports["errors"] = true
					if fieldType == "int" {
						validateParamsTemplates = append(validateParamsTemplates, fmt.Sprintf(minIntCode, field.Names[0].Name, parsedOpt[1], paramName, parsedOpt[1]))
					} else {
						validateParamsTemplates = append(validateParamsTemplates, fmt.Sprintf(minStrCode, field.Names[0].Name, parsedOpt[1], paramName, parsedOpt[1]))
					}
					continue OPTS_LOOP
				}

				if parsedOpt[0] == "max" {
					if _, err := strconv.Atoi(parsedOpt[1]); err != nil {
						log.Fatalln("max should be integer")
					}

					if fieldType == "int" {
						validateParamsTemplates = append(validateParamsTemplates, fmt.Sprintf(maxIntCode, field.Names[0].Name, parsedOpt[1], paramName, parsedOpt[1]))
					} else {
						log.Fatalln("max validator only for int params")
					}
					continue OPTS_LOOP
				}

			}
		}
	}

	if !isParamsStruct {
		return ""
	}

	funcBody := strings.Join(validateParamsTemplates, "")
	return fmt.Sprintf(validateFuncCode, structName, addSpace(funcBody, 1))
}

func getEnumCheck(enumList []string, fieldName string, paramName string, fieldType string) string {
	conditionList := []string{}
	if fieldType == "int" {
		condition := fmt.Sprintf("s.%v != %v", fieldName, 0)
		conditionList = append(conditionList, condition)
		for enumItem := range enumList {
			condition := fmt.Sprintf("s.%v != %v", fieldName, enumItem)
			conditionList = append(conditionList, condition)
		}
	} else if fieldType == "string" {
		condition := fmt.Sprintf("s.%v != \"%v\"", fieldName, "")
		conditionList = append(conditionList, condition)
		for _, enumItem := range enumList {
			condition := fmt.Sprintf("s.%v != \"%v\"", fieldName, enumItem)
			conditionList = append(conditionList, condition)
		}
	} else {
		log.Fatal("Only int and string types supported")
	}

	condStr := strings.Join(conditionList, " && ")

	return fmt.Sprintf(enumCheckCode, condStr, paramName, strings.Join(enumList, ", "))
}

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func addSpace(code string, num int) string {
	space := strings.Repeat("\t", num)
	codeStrings := strings.Split(code, "\n")
	for i := range codeStrings {
		if len(codeStrings[i]) != 0 {
			codeStrings[i] = space + codeStrings[i]
		}
	}
	return strings.Join(codeStrings, "\n")
}
