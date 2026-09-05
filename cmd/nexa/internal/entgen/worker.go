package entgen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

// RunWorker 仅供独立工作进程调用，Ent 内部执行不提供逐步骤取消
func RunWorker(ctx context.Context, input io.Reader) (err error) {
	var request workerRequest

	err = json.NewDecoder(input).Decode(&request)
	if err != nil {
		return
	}

	var directory string

	directory, err = os.Getwd()
	if err != nil {
		return
	}

	if !strings.HasPrefix(filepath.Base(directory), ".nexa-ent-") ||
		request.Bootstrap != filepath.Join(directory, "bootstrap") ||
		request.Output != filepath.Join(directory, "output") ||
		request.Overlay != filepath.Join(directory, "overlay.json") {
		err = fmt.Errorf("生成 Ent worker 必须在独立的临时目录中运行")
		return
	}

	var templates []*gen.Template

	for _, source := range []string{TemplateMeta, TemplateSoftDelete, TemplateUpsert} {
		var tmpl *gen.Template

		tmpl, err = gen.NewTemplate("nexa").Funcs(map[string]any{"softDeleteField": softDeleteField}).Parse(source)
		if err != nil {
			return
		}

		templates = append(templates, tmpl)
	}

	var latestOutput string
	config := gen.Config{
		Target:     request.Bootstrap,
		Package:    request.Package,
		IDType:     &field.TypeInfo{Type: field.TypeUint64},
		Templates:  templates,
		BuildFlags: []string{"-mod=readonly", "-overlay", request.Overlay},
		Hooks: []gen.Hook{func(next gen.Generator) gen.Generator {
			return gen.GenerateFunc(func(graph *gen.Graph) (err error) {
				if err = ctx.Err(); err != nil {
					return
				}

				err = validateSoftDelete(graph)
				if err != nil {
					return
				}

				var fresh string

				fresh, err = os.MkdirTemp(directory, "render-")
				if err != nil {
					return
				}

				graph.Target = fresh
				graph.Package = request.Package

				defer func() { graph.Target = request.Bootstrap }()

				err = next.Generate(graph)
				if err != nil {
					return
				}

				err = copyOutput(ctx, fresh, request.Bootstrap)
				if err != nil {
					return
				}

				graph.Target = request.Bootstrap
				_, err = gen.PrepareEnv(graph.Config)
				latestOutput = fresh

				return
			})
		}},
	}

	options := []entc.Option{entc.Storage("sql"), entc.FeatureNames(request.Features...)}
	for _, templateDirectory := range request.TemplateDirs {
		options = append(options, entc.TemplateDir(templateDirectory))
	}

	if err = ctx.Err(); err != nil {
		return
	}

	err = entc.Generate(request.Schema, &config, options...)
	if err == nil {
		err = copyOutput(ctx, latestOutput, request.Output)
	}

	return
}

func copyOutput(ctx context.Context, source, destination string) (err error) {
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) (err error) {
		if walkErr != nil {
			return walkErr
		}

		if err = ctx.Err(); err != nil || entry.IsDir() {
			return
		}

		if !entry.Type().IsRegular() {
			return fmt.Errorf("生成 Ent 输出不是普通文件：%s", path)
		}

		var relative string
		var content []byte

		relative, err = filepath.Rel(source, path)
		if err != nil {
			return
		}

		content, err = os.ReadFile(path)
		if err == nil {
			err = writeTemporary(filepath.Join(destination, relative), content)
		}

		return
	})

	return
}

func softDeleteField(node *gen.Type) *gen.Field {
	for _, candidate := range node.Fields {
		if candidate.Name == "deleted_at" {
			return candidate
		}
	}

	return nil
}

func validateSoftDelete(graph *gen.Graph) (err error) {
	for _, node := range graph.Nodes {
		candidate := softDeleteField(node)
		if candidate == nil {
			continue
		}

		if !candidate.IsTime() || !candidate.Optional || candidate.Immutable || candidate.HasGoType() {
			err = fmt.Errorf("%s.deleted_at 必须是可修改、Optional 的标准 time.Time 字段", node.Name)
			return
		}
	}

	return
}
