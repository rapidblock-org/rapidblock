package blockapply

type Config struct {
	Servers []Server `json:"servers" yaml:"servers"`
}

type Server struct {
	Name        string `json:"name" yaml:"name"`
	Mode        Mode   `json:"mode" yaml:"mode"`
	URI         string `json:"uri" yaml:"uri"`
	ClientToken string `json:"clientToken" yaml:"clientToken"`
}
