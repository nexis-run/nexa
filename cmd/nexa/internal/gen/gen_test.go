package gen

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/entgen"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

func TestPlanDAOBootstrapAndRepeat(t *testing.T) {
	generator := testGenerator(t)
	files, err := generator.PlanDAO([]string{"User"}, false, true)
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Contains(t, string(files[0].Content), "func NewUser(client *ent.Client)")
	require.Contains(t, string(files[1].Content), "dao.NewUser")
	require.Contains(t, string(files[1].Content), `wire.Struct(new(Dao), "User")`)
	require.NoDirExists(t, filepath.Dir(files[0].Path))
	require.NoDirExists(t, filepath.Dir(files[1].Path))

	var plan *fileplan.Plan

	plan, err = fileplan.New(files...)
	require.NoError(t, err)
	require.NoError(t, plan.Apply(context.Background()))

	files, err = generator.PlanDAO([]string{"User"}, false, true)
	require.NoError(t, err)

	plan, err = fileplan.New(files...)
	require.NoError(t, err)
	require.Empty(t, plan.Changes())
}

func TestPlanDAORejectsEntireInvalidBatch(t *testing.T) {
	generator := testGenerator(t)

	for _, names := range [][]string{{"User", "Missing"}, {"User", "USER"}, {"User", "User"}} {
		files, err := generator.PlanDAO(names, false, true)
		require.Error(t, err)
		require.Empty(t, files)
	}

	directory, err := generator.Config.GetDaoPath()
	require.NoError(t, err)
	require.NoDirExists(t, directory)

	require.NoError(t, os.WriteFile(filepath.Join(generator.Config.RootDir, "internal", "go.mod"), []byte("module nested\n"), 0644))
	var files []fileplan.File

	files, err = generator.PlanDAO([]string{"User"}, false, true)
	require.ErrorContains(t, err, "嵌套")
	require.Empty(t, files)
}

func TestPlanUsesDeclaredPackageAndCustomClient(t *testing.T) {
	generator := testGenerator(t)
	generator.Config.OrmClient = "database"
	generator.Config.DaoPath = "internal/type"
	directory, err := generator.Config.GetDaoPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(directory, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "package.go"), []byte("package storage\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "tools.go"), []byte("//go:build ignore\n\npackage main\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(generator.Config.RootDir, "go.mod"), []byte("module example.com/renamed\n"), 0644))
	require.NoError(t, generator.Config.Validate())

	var files []fileplan.File

	files, err = generator.PlanDAO([]string{"User"}, false, true)
	require.NoError(t, err)
	require.Contains(t, string(files[0].Content), "package storage")
	require.Contains(t, string(files[0].Content), "func NewUser()")
	require.Contains(t, string(files[0].Content), "(database).User")
	require.Contains(t, string(files[0].Content), "example.com/renamed/internal/infrastructure/ent")
	require.Contains(t, string(files[1].Content), "storage.NewUser")
}

func TestPlanUsesSharedSchemaDiscovery(t *testing.T) {
	for _, source := range []string{
		"import entalias `entgo.io/ent`\ntype User struct { entalias.Schema }\n",
		"import \"entgo.io/ent\"\ntype base struct { ent.Schema }; type User struct { base }\n",
		"import \"entgo.io/ent\"\ntype User struct { ent.View }\n",
		"import . \"entgo.io/ent\"\ntype User struct { Schema }\n",
	} {
		t.Run(source, func(t *testing.T) {
			generator := testGenerator(t)
			directory, err := generator.Config.GetEntPath()
			require.NoError(t, err)

			directory = filepath.Join(directory, "schema")
			require.NoError(t, os.WriteFile(filepath.Join(directory, "user.go"), []byte("package schema\n"+source), 0644))
			require.NoError(t, os.WriteFile(filepath.Join(directory, "ignored.go"), []byte("//go:build ignore\n\npackage main\ntype invalid =\n"), 0644))

			var entGenerator *entgen.EntGen

			entGenerator, err = entgen.New(generator.Config)
			require.NoError(t, err)
			var names []string

			names, err = entGenerator.SchemaNames()
			require.NoError(t, err)
			require.Equal(t, []string{"User"}, names)

			var files []fileplan.File

			files, err = generator.PlanDAO(names, false, false)
			require.NoError(t, err)
			require.Len(t, files, 1)
		})
	}
}

func testGenerator(t *testing.T) *Gen {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n\ngo 1.25\n"), 0644))

	cfg, err := base.NewDefaultConfig(filepath.Join(root, ".nexa.yaml"))
	require.NoError(t, err)

	var entDirectory string

	entDirectory, err = cfg.GetEntPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(entDirectory, "schema"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(entDirectory, "schema", "user.go"), []byte("package schema\nimport \"entgo.io/ent\"\ntype User struct { ent.Schema }\n"), 0644))

	var generator *Gen

	generator, err = New(cfg)
	require.NoError(t, err)

	return generator
}

func TestPlanRejectsNamesAndDeclarations(t *testing.T) {
	generator := testGenerator(t)

	for _, names := range [][]string{
		{"Rider_Test"}, {"Rider_Linux"}, {"Rider_AMD64"}, {"Rider", "GetRider"},
	} {
		files, err := generator.PlanEchoContext(names, true)
		require.Error(t, err)
		require.Empty(t, files)
	}

	directory, err := generator.Config.GetAbsPath(generator.Config.EchoctxPath)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(directory, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "existing.go"), []byte("package app\nvar RiderContext any\n"), 0644))

	_, err = generator.PlanEchoContext([]string{"Rider"}, true)
	require.ErrorContains(t, err, "RiderContext")
	require.NoFileExists(t, filepath.Join(directory, "rider.go"))
}
