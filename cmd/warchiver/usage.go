package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

type FlagWithShortcut struct {
	Key      string
	Sort     string
	Flag     *flag.Flag
	Shortcut *flag.Flag
}

func FlagSetUsage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "%v %s\n", AppName(), fs.Name())

	flagMap := map[string]*FlagWithShortcut{} // usage => flag+shortcut struct
	fs.VisitAll(func(f *flag.Flag) {
		if fg, exists := flagMap[f.Usage]; exists {
			fg.Shortcut = f
			flagMap[f.Usage] = fg
		} else {
			flagMap[f.Usage] = &FlagWithShortcut{
				Flag: f,
				Key:  f.Usage,
				Sort: f.Name,
			}
		}
	})

	if len(flagMap) > 0 {
		fmt.Fprintf(os.Stderr, "flags:\n")
	}

	flagList := make([]*FlagWithShortcut, 0, len(flagMap))
	for _, v := range flagMap {
		flagList = append(flagList, v)
	}

	sort.Slice(flagList, func(i, j int) bool {
		return flagList[i].Sort < flagList[j].Sort
	})

	for _, fs2 := range flagList {
		line1 := "-" + fs2.Flag.Name
		if fs2.Shortcut != nil {
			line1 = line1 + ",-" + fs2.Shortcut.Name
		}
		typeStr, line2 := flag.UnquoteUsage(fs2.Flag)
		if fs2.Flag.DefValue != "" {
			line2 = line2 + " (default \"" + fs2.Flag.DefValue + "\")"
		}

		fmt.Fprintf(fs.Output(), "  %v %v\n      %v\n", line1, typeStr, line2)
	}
}
