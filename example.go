package saz

import (
	"fmt"
)

func test(){
	urlMatchList := []string{
		"^https:\\/\\/api.test.net\\/tst\\/s\\/v5\\/book\\/classify\\/bookList",
	}

	filepath := "./input/book_list_page.saz"
	result, err := ParseFile(filepath, urlMatchList)
	if err != nil {
		return
	}
	fmt.Println(len(result.Requests))
}
