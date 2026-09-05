package gen

import (
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

// Gen 根据显式配置渲染待写文件
type Gen struct {
	Config *base.Config
}

func New(cfg *base.Config) (generator *Gen, err error) {
	if cfg == nil {
		err = errors.New("生成器配置不能为空")
		return
	}

	_, err = cfg.ResolveModule()
	if err != nil {
		return
	}

	_, err = parsedTemplates()
	if err != nil {
		return
	}

	generator = &Gen{Config: cfg}

	return
}

func (generator *Gen) PlanEchoContext(names []string, force bool) (files []fileplan.File, err error) {
	var directory, packageName string

	directory, packageName, err = generator.preparePackage(generator.Config.EchoctxPath, names)
	if err != nil {
		return
	}

	for _, name := range names {
		var content []byte

		content, err = renderGo("echoctx.tmpl", &base.EchoCtxTemplateVariables{
			Package: packageName,
			Name:    name,
		})
		if err != nil {
			files = nil
			return
		}

		files = append(files, fileplan.File{Path: outputPath(directory, name), Content: content, Overwrite: force})
	}

	return
}

func (generator *Gen) preparePackage(configuredPath string, names []string) (directory, packageName string, err error) {
	err = validateNames(names)
	if err != nil {
		return
	}

	_, err = generator.Config.ResolveModule()
	if err != nil {
		return
	}

	directory, err = generator.Config.GetAbsPath(configuredPath)
	if err != nil {
		return
	}

	_, err = generator.Config.ResolvePackagePath(directory)
	if err != nil {
		return
	}

	filenames := make([]string, 0, len(names))

	for _, name := range names {
		filenames = append(filenames, filepath.Base(outputPath(directory, name)))
	}

	err = preflightFiles(directory, filenames)
	if err != nil {
		return
	}

	packageName, err = base.ResolvePackageName(directory)

	return
}

func validateNames(names []string) error {
	if len(names) == 0 {
		return base.ErrNameRequired
	}

	for index, name := range names {
		if !base.StringIsExportedIdentifier(name) {
			return fmt.Errorf("%w：%s", base.ErrInvalidExportedName, name)
		}

		for _, previous := range names[:index] {
			if strings.EqualFold(previous, name) {
				return fmt.Errorf("名称重复或生成文件名冲突：%s、%s", previous, name)
			}
		}
	}

	return nil
}

func preflightFiles(directory string, filenames []string) error {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	for _, filename := range filenames {
		for _, entry := range entries {
			if !strings.EqualFold(entry.Name(), filename) {
				continue
			}

			if entry.Name() != filename {
				return fmt.Errorf("目标文件与已有文件仅大小写不同：%s", filepath.Join(directory, entry.Name()))
			}

			if entry.IsDir() {
				return fmt.Errorf("目标文件路径是目录：%s", filepath.Join(directory, filename))
			}
		}
	}

	return nil
}

func outputPath(directory, name string) string {
	return filepath.Join(directory, strings.ToLower(name)+".go")
}

func renderGo(name string, variables any) (content []byte, err error) {
	content, err = RenderTemplate(name, variables)
	if err != nil {
		return
	}

	content, err = format.Source(content)
	if err != nil {
		err = fmt.Errorf("模板 %s 未生成有效 Go 代码：%w", name, err)
	}

	return
}
