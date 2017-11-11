# pknulms

[![GoDoc](https://godoc.org/github.com/hallazzang/pknulms?status.svg)](https://godoc.org/github.com/hallazzang/pknulms) [![Go Report Card](https://goreportcard.com/badge/github.com/hallazzang/pknulms)](https://goreportcard.com/report/github.com/hallazzang/pknulms)

Go LMS client for Pukyong National University.

## Getting Started

### Installation

```bash
$ go get github.com/hallazzang/pknulms
```

### Example

Here's a brief example:
```go
package main

import (
	"fmt"
	"os"

	"github.com/hallazzang/pknulms"
)

func main() {
	client := pknulms.MustNewClient()

	if !client.MustLogin("YOUR_STUDENT_NO", "YOUR_PASSWORD") { // You should replace these values
        panic("Login failed.")
	}

	for i, n := range client.MustGetNotificationsByPage(1) {
		fmt.Printf("%2d: %s\n", i+1, n.Title)
	}
}
```

It should print recent notifications like this:
```
 1: 수업자료_14장-15장
 2: 8. 6장 큐 연습문제
 3: 6.  7장 트리 (Tree)
 4: Ch.10 Input Output Organization
 5: Ch.09 Memory Organization
 6: Ch.08 Computer Arithmetic
 7: Ch.07 Microsequencer Control Unit Design
 8: 7. 미로문제 풀이 프로그래밍
 9: 6. 5장 스택 연습문제
10: 강의자료6
11: 실습제출5
12: 5. 6장 큐 (Updated 10/31, 11/2)
13: 4. 5장 스택(Stack)
14: 강의자료5
15: 실습제출4
16: 강의자료4
17: 실습제출3
18: 5. 4장 리스트 연습문제
19: 4. 배열을 이용한 리스트 테스트 프로그래밍
20: 강의자료3
```
