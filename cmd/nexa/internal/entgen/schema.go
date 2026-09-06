package entgen

// SchemaNames 静态读取嵌入 Ent Schema 或 View 的类型，不执行用户代码
func (eng *EntGen) SchemaNames() ([]string, error) {
	return eng.cfg.SchemaNames()
}
