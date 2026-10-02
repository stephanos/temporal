package dynamicconfig

type Setting struct{}

func NewGlobalBoolSetting(string, bool, string) Setting { return Setting{} }

func NewNamespaceIntSetting(string, int, string) Setting { return Setting{} }
