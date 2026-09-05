package base

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeConfigFixture(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func moduleFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "project with spaces")
	writeConfigFixture(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.26.2\n")
	physical, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)

	return physical
}

func TestLoadConfigStrictYAML(t *testing.T) {
	root := moduleFixture(t)
	path := filepath.Join(root, "settings", "project config.yaml")

	for _, content := range []string{
		"",
		"null\n",
		"[]\n",
		"unexpected: true\n",
		"entPath: internal/ent\nentPath: internal/other\n",
		"di:\n  unexpected: true\n",
		"entPath: null\n",
		"di: null\n",
		"{}\n---\n{}\n",
		"{}\n---\n",
		"entPath: ''\n",
		"daoPath: internal/type\n",
		"di:\n  daoStructName: bad-name\n",
		"di:\n  daoProviderSetVar: _\n",
		"di:\n  path: internal/di/di_test.go\n",
		"ormclient: 'ent.Database; panic(1)'\n",
	} {
		t.Run(content, func(t *testing.T) {
			writeConfigFixture(t, path, content)
			_, err := LoadConfig(LoadOptions{ConfigPath: path, Explicit: true})
			require.Error(t, err)
		})
	}

	writeConfigFixture(t, path, "entPath: internal/store\normclient: ent.Database\n")
	config, err := LoadConfig(LoadOptions{ConfigPath: path, Explicit: true})
	require.NoError(t, err)
	require.Equal(t, root, config.RootDir)
	require.Equal(t, "ent.Database", config.OrmClient)

	var module, packagePath string

	module, err = config.ResolveModule()
	require.NoError(t, err)
	require.Equal(t, "example.com/project", module)

	packagePath, err = config.ResolvePackagePath(config.EntPath)
	require.NoError(t, err)
	require.Equal(t, "example.com/project/internal/store", packagePath)
}

func TestConfigDiscoveryAndMissingRules(t *testing.T) {
	root := moduleFixture(t)
	configPath := filepath.Join(root, DefaultConfigFile)
	writeConfigFixture(t, configPath, "entPath: internal/store\n")

	child := filepath.Join(root, "nested", "commands")
	require.NoError(t, os.MkdirAll(child, 0755))
	t.Chdir(child)

	config, err := LoadConfig(LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, configPath, config.GetConfigFilePath())
	require.True(t, config.ConfigExists())

	_, err = LoadConfig(LoadOptions{ConfigPath: DefaultConfigFile, Explicit: true})
	require.ErrorIs(t, err, os.ErrNotExist)

	var missing, isolated *Config

	missing, err = LoadConfig(LoadOptions{ConfigPath: "missing.yaml", Explicit: true, AllowMissing: true})
	require.NoError(t, err)
	require.False(t, missing.ConfigExists())
	require.Empty(t, missing.OrmClient)

	_, err = LoadConfig(LoadOptions{ConfigPath: "missing.yaml"})
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = LoadConfig(LoadOptions{Explicit: true})
	require.Error(t, err)

	writeConfigFixture(t, filepath.Join(child, "go.mod"), "module localmodule\n")

	isolated, err = LoadConfig(LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(child, DefaultConfigFile), isolated.GetConfigFilePath())
	require.False(t, isolated.ConfigExists())

	var module, resolved string

	module, err = isolated.ResolveModule()
	require.NoError(t, err)
	require.Equal(t, "localmodule", module)

	t.Chdir(t.TempDir())

	resolved, err = config.GetAbsPath(config.EntPath)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "internal/store"), resolved)
}

func TestNewDefaultConfigDoesNotDecodeTarget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings", "broken.yaml")
	writeConfigFixture(t, path, "[ malformed")
	config, err := NewDefaultConfig(path)
	require.NoError(t, err)
	require.True(t, config.ConfigExists())
	require.Empty(t, config.OrmClient)

	_, err = config.ResolveModule()
	require.ErrorIs(t, err, os.ErrNotExist)

	var encoded []byte

	encoded, err = config.MarshalYAMLBytes()
	require.NoError(t, err)
	require.Contains(t, string(encoded), "ormclient: \"\"")
	require.NotContains(t, string(encoded), "rootdir")
	require.NotContains(t, string(encoded), root)

	encoded, err = json.Marshal(config)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"entPath"`)
	require.NotContains(t, string(encoded), `"RootDir"`)
}

func TestConfigRejectsUnsafeOutputs(t *testing.T) {
	for _, scenario := range []string{"outside", "symlink ancestor", "symlink target", "di symlink", "nested module", "di nested module", "directory as file"} {
		t.Run(scenario, func(t *testing.T) {
			root := moduleFixture(t)
			config, err := NewDefaultConfig(filepath.Join(root, DefaultConfigFile))
			require.NoError(t, err)

			switch scenario {
			case "outside":
				config.EntPath = "../outside"
			case "symlink ancestor":
				require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(root, "escape")))

				config.EntPath = "escape/ent"
			case "symlink target":
				require.NoError(t, os.MkdirAll(filepath.Join(root, "actual"), 0755))
				require.NoError(t, os.Symlink(filepath.Join(root, "actual"), filepath.Join(root, "ent")))

				config.EntPath = "ent"
			case "di symlink":
				writeConfigFixture(t, filepath.Join(root, "actual.go"), "package di\n")
				require.NoError(t, os.Symlink(filepath.Join(root, "actual.go"), filepath.Join(root, "di.go")))

				config.DI.Path = "di.go"
			case "nested module":
				writeConfigFixture(t, filepath.Join(root, "nested/go.mod"), "module nested\n")

				config.DaoPath = "nested/dao"
			case "di nested module":
				writeConfigFixture(t, filepath.Join(root, "nested/go.mod"), "module nested\n")

				config.DI.Path = "nested/di.go"
			case "directory as file":
				require.NoError(t, os.MkdirAll(filepath.Join(root, "di.go"), 0755))

				config.DI.Path = "di.go"
			}

			require.Error(t, config.Validate())
		})
	}
}

func TestPhysicalRootAndModuleChanges(t *testing.T) {
	root := moduleFixture(t)
	alias := filepath.Join(t.TempDir(), "project-alias")
	require.NoError(t, os.Symlink(root, alias))
	config, err := NewDefaultConfig(filepath.Join(alias, DefaultConfigFile))
	require.NoError(t, err)
	require.Equal(t, root, config.RootDir)
	require.Equal(t, filepath.Join(root, DefaultConfigFile), config.GetConfigFilePath())

	_, err = config.ResolveModule()
	require.NoError(t, err)

	writeConfigFixture(t, filepath.Join(root, "internal/go.mod"), "module nested\n")

	_, err = config.ResolveModule()
	require.ErrorContains(t, err, "嵌套")

	config.EntPath = "ent"
	config.DaoPath = "dao"
	config.EchoctxPath = "app"
	config.DI.Path = "di/di.go"
	writeConfigFixture(t, filepath.Join(root, "go.mod"), "module example.com/changed\n")
	var module, packagePath string

	module, err = config.ResolveModule()
	require.NoError(t, err)
	require.Equal(t, "example.com/changed", module)

	packagePath, err = config.ResolvePackagePath(config.EntPath)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(packagePath, module+"/"))
}
