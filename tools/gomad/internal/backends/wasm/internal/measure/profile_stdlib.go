package measure

import "strings"

func init() {
	for _, name := range strings.Fields("archive bufio bytes cmp compress container context crypto database debug embed encoding errors expvar flag fmt hash html image index internal io iter log maps math mime net os path plugin reflect regexp runtime slices sort strconv strings structs sync syscall testing text time unicode unique unsafe weak") {
		guestStdlibTopLevel[name] = true
	}
}
