package entgen

import (
	"path/filepath"

	"nexis.run/nexa/cmd/nexa/internal/schema"
)

// SchemaNames 静态读取嵌入 Ent Schema 或 View 的类型，不执行用户代码
func (eng *EntGen) SchemaNames() (names []string, err error) {
	var entPath string

	entPath, err = eng.cfg.GetEntPath()
	if err != nil {
		return
	}

	names, err = schema.Names(filepath.Join(entPath, "schema"))

	return
}
