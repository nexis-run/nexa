// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-25, by liasica

package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDIProvider(t *testing.T) {
	dp, err := NewDaoProvider("../../../../tests/di.gofile", "Dao", "daoProviderSet", "auroraride.com/oos/internal/infrastructure/dao")
	require.NoError(t, err)

	dp.AddField("Agreement", "Brand", "City", "Manager", "System")

	var b []byte

	b, err = dp.Generate()
	require.NoError(t, err)
	require.Greater(t, strings.Index(string(b), "// 集成所有第三方或外部服务"), strings.Index(string(b), "var daoProviderSet"))

	t.Logf("Generated DI Provider:\n%s", string(b))
}

func TestDaoProviderPreservesCustomEntries(t *testing.T) {
	diPath := filepath.Join(t.TempDir(), "di.go")
	content := []byte(`package di

import (
    wiring "github.com/google/wire"
    data "example.com/project/dao"
)

type Custom struct{}

type Dao struct {
	Custom *Custom
	Existing *data.ExistingDao
}


func newCustom() *Custom {
	return &Custom{}
}

func newExistingWithOptions() *data.ExistingDao {
    return data.NewExisting()
}

var daoProviderSet = wiring.NewSet(
	newCustom,
	newExistingWithOptions,
	wiring.Struct(&Dao{}, "Custom", "Existing"),
)
`)
	require.NoError(t, os.WriteFile(diPath, content, 0644))

	provider, err := NewDaoProvider(diPath, "Dao", "daoProviderSet", "example.com/project/dao")
	require.NoError(t, err)

	provider.AddField("User")

	var generated []byte

	generated, err = provider.Generate()
	require.NoError(t, err)

	result := string(generated)
	require.Regexp(t, `Custom\s+\*Custom`, result)
	require.Regexp(t, `User\s+\*data\.UserDao`, result)
	require.Contains(t, result, "newCustom")
	require.Contains(t, result, "newExistingWithOptions,")
	require.Contains(t, result, "data.NewUser")
	require.Contains(t, result, `wiring.Struct(&Dao{}, "Custom", "Existing", "User")`)
	require.Equal(t, 1, strings.Count(result, "data.NewUser"))
	require.Equal(t, 1, strings.Count(result, "data.NewExisting"))

	require.NoError(t, os.WriteFile(diPath, generated, 0644))

	provider, err = NewDaoProvider(diPath, "Dao", "daoProviderSet", "example.com/project/dao")
	require.NoError(t, err)

	provider.AddField("User", "Existing")

	var repeated []byte

	repeated, err = provider.Generate()
	require.NoError(t, err)
	require.Equal(t, generated, repeated)
}

func TestDaoProviderUsesActualPackageName(t *testing.T) {
	content := []byte(`package di
import (
	"github.com/google/wire"
	"example.com/project/dao"
)
type Dao struct { Existing *storage.ExistingDao }
var daoProviderSet = wire.NewSet(customExisting, wire.Struct(new(Dao), "Existing"))
`)
	provider, err := ParseDaoProvider(DaoProviderConfig{
		Path:         "di.go",
		ImportPath:   "example.com/project/dao",
		PackageName:  "storage",
		TypeName:     "Dao",
		VariableName: "daoProviderSet",
	}, content)
	require.NoError(t, err)

	provider.AddField("User")
	var generated []byte

	generated, err = provider.Generate()
	require.NoError(t, err)
	require.Contains(t, string(generated), "storage.NewUser")
	require.Contains(t, string(generated), "customExisting")
	require.Contains(t, string(generated), `wire.Struct(new(Dao), "Existing", "User")`)
}

func TestDaoProviderCompletesExistingWireField(t *testing.T) {
	source := []byte(`package di
import (
    "github.com/google/wire"
    "example.com/project/dao"
)
type Dao struct { User *dao.UserDao }
var daoProviderSet = wire.NewSet(customUser, wire.Struct(new(Dao)))
`)
	provider, err := ParseDaoProvider(DaoProviderConfig{
		Path: "di.go", ImportPath: "example.com/project/dao", PackageName: "dao",
		TypeName: "Dao", VariableName: "daoProviderSet",
	}, source)
	require.NoError(t, err)
	provider.AddField("User")
	var generated []byte

	generated, err = provider.Generate()
	require.NoError(t, err)
	require.Contains(t, string(generated), `wire.Struct(new(Dao), "User")`)
	require.Contains(t, string(generated), "customUser")
	require.NotContains(t, string(generated), "dao.NewUser")
}
