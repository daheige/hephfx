package main

import (
	"bufio"
	"flag"
	"html/template"
	"io"
	"log"
	"os"
	"strings"
)

var (
	validatorLogDir string
	pbDir           string
)

var validatorTpl = `package pb

// Code generated by validator_gen tools.DO NOT EDIT.
import (
	validator "github.com/go-playground/validator/v10"
)

var validate = validator.New()`

var validatorReqTpl = `// Validate {{.Name}} interceptor validator.
func (r *{{.Name}}) Validate() error {
	return validate.Struct(r)
}`

// Field 模版字段结构体
type Field struct {
	Name string
}

func init() {
	flag.StringVar(&validatorLogDir, "validator_log_dir", "./", "validator log dir")
	flag.StringVar(&pbDir, "pb_dir", "./pb", "go pb dir")
	flag.Parse()

	if pbDir == "" || validatorLogDir == "" {
		log.Fatalln("param error")
	}

	pbDir = strings.TrimRight(pbDir, "/")
}

func main() {
	validators := getAllReqValidatorNames()
	if len(validators) == 0 {
		return
	}

	fd, err := os.Create(pbDir + "/validator.go")
	if err != nil {
		log.Fatalln("create validator.go error: ", err)
	}

	defer fd.Close()
	fd.Write([]byte(validatorTpl))
	fd.Write([]byte("\n"))

	tmpl, err := template.New("validator_req").Parse(validatorReqTpl)
	if err != nil {
		log.Fatalln("validator_req template error: ", err)
	}

	// 多个拦截器生成
	fd, err = os.Create(pbDir + "/validator_req.go")
	if err != nil {
		log.Fatalln("create validator_req.go error: ", err)
	}

	defer fd.Close()

	fd.Write([]byte("package pb\n\n"))
	fd.Write([]byte("// Code generated by validator_gen tools.DO NOT EDIT.\n\n"))

	for k := range validators {
		c := Field{
			Name: validators[k],
		}

		err = tmpl.Execute(fd, c)
		if err != nil {
			log.Printf("generate validator: %s \n", validators[k])
			continue
		}

		fd.Write([]byte("\n\n"))
	}

	log.Println("generate validator success")
}

func getAllReqValidatorNames() []string {
	// 读取文件的内容
	file, err := os.Open(validatorLogDir + "/validator.log")
	if err != nil {
		log.Println("open file err:", err.Error())
		return nil
	}

	// 处理结束后关闭文件
	defer file.Close()

	arr := make([]string, 0, 20)

	// 使用bufio读取
	r := bufio.NewReader(file)
	for {
		// 以分隔符形式读取,比如此处设置的分割符是\n,则遇到\n就返回,且包括\n本身 直接返回字节数数组
		data, err := r.ReadBytes('\n')

		// 读取到末尾退出
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("read err", err.Error())
			break
		}

		str := string(data)
		str = strings.Trim(str, "\n")
		if str != "" {
			// 打印出内容
			log.Println(str)
			s := strings.Split(str, "=")

			// log.Println("s len = ", len(s))
			// log.Println("s = ", s)

			if len(s) == 2 {
				arr = append(arr, strings.TrimSpace(s[1]))
			}
		}
	}

	log.Println("req validator: ", arr)

	return arr
}
