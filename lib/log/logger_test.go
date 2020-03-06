package log

import (
	"fmt"
	golog "log"
	"strings"
)

func ExamplePrintf() {
	buffer := strings.Builder{}
	golog.SetOutput(&buffer)
	golog.SetFlags(0)

	SetLevel("info")
	Printf("regular message")
	Printf("[critical] critical error message")
	Printf("[error] regular error")
	Printf("[warning] something that may require some attention")
	Printf("[info] message about what the program is doing")
	Printf("[debug] debugging information")
	Printf("[trace] trace deep internals")

	fmt.Print(buffer.String())
	// Output:
	// regular message
	// [critical] critical error message
	// [error] regular error
	// [warning] something that may require some attention
	// [info] message about what the program is doing
}
