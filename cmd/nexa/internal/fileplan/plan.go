// Package fileplan 预检并应用代码生成产生的文件变更
package fileplan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type File struct {
	Path      string
	Content   []byte
	Overwrite bool
	Remove    bool
}

type Change struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

type plannedFile struct {
	File
	original []byte
	mode     os.FileMode
	existed  bool
	staged   string
}

type Plan struct {
	files   []plannedFile
	changes []Change
}

// New 在写入前检查所有目标，忽略内容相同的文件
func New(files ...File) (plan *Plan, err error) {
	plan = &Plan{changes: make([]Change, 0)}
	seen := make(map[string]bool)

	for _, file := range files {
		file.Path, err = normalizePath(file.Path)
		if err != nil {
			return
		}

		key := strings.ToLower(file.Path)
		if seen[key] {
			err = fmt.Errorf("生成目标重复：%s", file.Path)
			return
		}

		seen[key] = true

		err = validatePath(file.Path)
		if err != nil {
			return
		}

		var current plannedFile

		current, err = inspect(file)
		if err != nil {
			return
		}

		if file.Remove && !current.existed || !file.Remove && current.existed && bytes.Equal(current.original, file.Content) {
			continue
		}

		if current.existed && !file.Overwrite {
			err = fmt.Errorf("文件已存在：%s，覆盖时请使用 --force", file.Path)
			return
		}

		action := "create"

		if file.Remove {
			action = "delete"
		} else if current.existed {
			action = "update"
		}

		current.Content = bytes.Clone(file.Content)
		plan.files = append(plan.files, current)
		plan.changes = append(plan.changes, Change{Path: file.Path, Action: action})
	}

	return
}

func (plan *Plan) Changes() []Change {
	return slices.Clone(plan.changes)
}

// Apply 先暂存全部内容，再逐个替换；失败时恢复已写入的文件
// 多文件替换不提供跨进程原子性，发生并发修改时停止并报告
func (plan *Plan) Apply(ctx context.Context) (err error) {
	if err = ctx.Err(); err != nil {
		return
	}

	for _, file := range plan.files {
		err = file.checkUnchanged()
		if err != nil {
			return
		}
	}

	var createdDirs []string

	defer func() {
		for _, file := range plan.files {
			if file.staged != "" {
				_ = os.Remove(file.staged)
			}
		}

		if err != nil {
			for _, directory := range createdDirs {
				_ = os.Remove(directory)
			}
		}
	}()

	for index := range plan.files {
		if err = ctx.Err(); err != nil {
			return
		}

		file := &plan.files[index]
		if file.Remove {
			continue
		}

		var directories []string

		directories, err = makeParentDirs(filepath.Dir(file.Path))
		createdDirs = append(directories, createdDirs...)

		if err != nil {
			return
		}

		file.staged, err = stage(file.Path, file.Content, file.mode)
		if err != nil {
			return
		}
	}

	for index := range plan.files {
		file := &plan.files[index]

		err = ctx.Err()
		if err == nil {
			err = file.checkUnchanged()
		}

		if err == nil {
			if file.Remove {
				err = os.Remove(file.Path)
			} else if file.existed {
				err = os.Rename(file.staged, file.Path)
			} else {
				// 新建文件使用排他链接，不覆盖同时创建的目标
				err = os.Link(file.staged, file.Path)
			}
		}

		if err != nil {
			err = errors.Join(fmt.Errorf("应用文件变更失败：%w", err), plan.rollback(index))
			return
		}
	}

	return
}

func inspect(file File) (current plannedFile, err error) {
	current.File = file
	current.mode = 0644
	var info os.FileInfo

	info, err = os.Lstat(file.Path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
		return
	}

	if err != nil {
		return
	}

	if !info.Mode().IsRegular() {
		err = fmt.Errorf("生成目标不是普通文件：%s", file.Path)
		return
	}

	current.existed = true
	current.mode = info.Mode().Perm()
	current.original, err = os.ReadFile(file.Path)

	return
}

func (file plannedFile) checkUnchanged() (err error) {
	err = validatePath(file.Path)
	if err != nil {
		return
	}

	var current plannedFile

	current, err = inspect(file.File)
	if err != nil {
		return
	}

	if file.existed != current.existed || file.mode != current.mode || !bytes.Equal(file.original, current.original) {
		err = fmt.Errorf("目标文件在生成期间发生变化：%s，请重新执行", file.Path)
	}

	return
}

func validatePath(path string) (err error) {
	for current := path; ; current = filepath.Dir(current) {
		var info os.FileInfo

		info, err = os.Lstat(current)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}

		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			err = fmt.Errorf("生成路径不能包含符号链接：%s", current)
			return
		}

		if current != path && err == nil && !info.IsDir() {
			err = fmt.Errorf("生成路径的父级不是目录：%s", current)
			return
		}

		if filepath.Dir(current) == current {
			err = nil
			return
		}
	}
}

func normalizePath(path string) (normalized string, err error) {
	normalized, err = filepath.Abs(path)
	if err != nil {
		return
	}

	for ancestor := filepath.Dir(normalized); ; ancestor = filepath.Dir(ancestor) {
		var resolved string

		resolved, err = filepath.EvalSymlinks(ancestor)
		if err == nil {
			var relative string

			relative, err = filepath.Rel(ancestor, normalized)
			if err == nil {
				normalized = filepath.Join(resolved, relative)
			}

			return
		}

		if !errors.Is(err, os.ErrNotExist) || filepath.Dir(ancestor) == ancestor {
			return
		}
	}
}

func makeParentDirs(directory string) (created []string, err error) {
	for current := directory; ; current = filepath.Dir(current) {
		_, err = os.Lstat(current)
		if err == nil {
			break
		}

		if !errors.Is(err, os.ErrNotExist) {
			return
		}

		created = append(created, current)
	}

	err = os.MkdirAll(directory, 0755)

	return
}

func stage(path string, content []byte, mode os.FileMode) (temporary string, err error) {
	var file *os.File

	file, err = os.CreateTemp(filepath.Dir(path), ".nexa-*")
	if err != nil {
		return
	}

	temporary = file.Name()

	defer func() {
		_ = file.Close()

		if err != nil {
			_ = os.Remove(temporary)
		}
	}()

	_, err = file.Write(content)
	if err == nil {
		err = file.Chmod(mode)
	}

	if err == nil {
		err = file.Close()
	}

	return
}

func (plan *Plan) rollback(count int) (err error) {
	for index := count - 1; index >= 0; index-- {
		err = errors.Join(err, plan.files[index].restore())
	}

	return
}

func (file plannedFile) restore() (err error) {
	err = validatePath(file.Path)
	if err != nil {
		return
	}

	var current plannedFile

	current, err = inspect(file.File)
	if err != nil {
		return
	}

	if file.Remove && current.existed || !file.Remove && (!current.existed || current.mode != file.mode || !bytes.Equal(current.original, file.Content)) {
		err = fmt.Errorf("目标文件已被其他进程修改，未回滚：%s", file.Path)
		return
	}

	if !file.existed {
		err = os.Remove(file.Path)
		return
	}

	var temporary string

	temporary, err = stage(file.Path, file.original, file.mode)
	if err != nil {
		return
	}

	defer func() { _ = os.Remove(temporary) }()

	if file.Remove {
		err = os.Link(temporary, file.Path)
	} else {
		err = os.Rename(temporary, file.Path)
	}

	return
}
