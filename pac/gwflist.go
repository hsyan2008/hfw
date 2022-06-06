package pac

import "github.com/hsyan2008/gfwlist4go/gfwlist"

func LoadGwflist() (err error) {
	//通过代理
	blacklist, err := gfwlist.BlankList()
	if err != nil {
		return
	}
	for _, v := range blacklist {
		add(v, true)
	}

	return
}
