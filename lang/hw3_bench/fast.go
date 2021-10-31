package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/mailru/easyjson"

	"github.com/adromaryn/mailru-go/lang/hw3_bench/structs"
)

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	r := bufio.NewReader(file)

	emailSplitterRegex := regexp.MustCompile("@")
	androidRegex := regexp.MustCompile("Android")
	msieRegex := regexp.MustCompile("MSIE")
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	users := make([]structs.UserData, 0)
	for {
		s, err := r.ReadBytes('\n')
		if err != nil && err != io.EOF {
			panic(err)
		}
		if len(s) == 0 && err == io.EOF {
			break
		}
		user := &structs.UserData{}
		// fmt.Printf("%v %v\n", err, line)
		err2 := easyjson.Unmarshal([]byte(s), user)
		if err2 != nil {
			panic(err)
		}
		users = append(users, *user)

		if err == io.EOF {
			break
		}
	}

	for i, user := range users {

		isAndroid := false
		isMSIE := false

		browsers := user.Browsers
		for _, browser := range browsers {
			if ok := androidRegex.MatchString(browser); ok {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		for _, browser := range browsers {
			if ok := msieRegex.MatchString(browser); ok {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := emailSplitterRegex.ReplaceAllString(user.Email, " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
